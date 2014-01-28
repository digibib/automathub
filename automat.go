package main

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
	"strings"

	"github.com/gorilla/websocket"
)

// state of an automats user iterface
type uiState uint8

const (
	// possible UI states:
	uiWAITING uiState = iota
	uiCHECKIN
	uiCHECKOUT
	uiSTATUS
	uiERROR
)

// Automat is a state machine for the automats. It recieves communcations
// from RFID service, User Interface and communicates with the SIP server.
type Automat struct {
	State         uiState
	Authenticated bool   // logged in or not
	IP            string // remote address of the automat
	Dept          string // department (SIP: institution id)
	Patron        string // patron username

	// TODO
	// Keep track of transactions, and send to RFIDservice for printout
	// upon request. Clear on logout.
	Checkins  []string
	Checkouts []string

	// SIP connection (via TCP)
	SIPConn net.Conn

	// Communication with the RFID service (via TCP)
	RFIDconn net.Conn
	FromRFID chan []byte
	ToRFID   chan []byte

	// User inteface communication (via Websocket)
	ws     *websocket.Conn
	ToUI   chan []byte
	FromUI chan []byte

	Quit chan bool // For closing down the state machine
}

// return a new Automat (ceated upon receiving a tcp connection)
func newAutomat(c net.Conn) *Automat {
	return &Automat{
		State:    uiWAITING,
		IP:       c.RemoteAddr().String(),
		RFIDconn: c,
		FromRFID: make(chan []byte),
		ToRFID:   make(chan []byte),
		ToUI:     make(chan []byte),
		FromUI:   make(chan []byte),
		Quit:     make(chan bool),
	}
}

// run the Automat state machine & message handler
func (a *Automat) run() {

	for {
		select {
		case msg := <-a.FromRFID:
			log.Println("<- RFID:", strings.TrimRight(string(msg), "\n"))
			rfidMsg, err := parseRFIDRequest(msg)
			if err != nil {
				log.Println("ERROR", err.Error())
				// TODO respond to RFIDservise? and what?
				break
			}
			//log.Printf("DEBUG %+v", rfidMsg)
			var sipRes *UIResponse
			switch a.State {
			case uiCHECKIN:
				sipRes, err = DoSIPCall(sipPool, sipFormMsgCheckin(a.Dept, rfidMsg.Barcode), checkinParse)
				sipRes.Action = "CHECKIN"
			case uiCHECKOUT:
				sipRes, err = DoSIPCall(sipPool, sipFormMsgCheckout(a.Patron, rfidMsg.Barcode), checkoutParse)
				sipRes.Action = "CHECKOUT"
			default:
				log.Println("ERROR", "state: %+v | rfidmessage: %v", a.State, rfidMsg)
				break
			}
			if err != nil {
				log.Println(err)
				// TODO now what?
				break
			}
			bRes, err := json.Marshal(sipRes)
			if err != nil {
				a.ToUI <- ErrorResponse(err)
				break
			}
			a.ToUI <- bRes
		case msg := <-a.FromUI:
			log.Println("<- UI", strings.TrimRight(string(msg), "\n"))
			var uiMsg UIRequest
			err := json.Unmarshal(msg, &uiMsg)
			if err != nil {
				a.ToUI <- ErrorResponse(err)
			} else {
				switch uiMsg.Action {
				case "LOGIN":
					authRes, err := DoSIPCall(sipPool, sipFormMsgAuthenticate(a.Dept, uiMsg.Username, uiMsg.PIN), authParse)
					if err != nil {
						a.ToUI <- ErrorResponse(err)
						break
					}

					bRes, err := json.Marshal(authRes)
					if err != nil {
						a.ToUI <- ErrorResponse(err)
						break
					}
					a.Authenticated = authRes.Authenticated
					if a.Authenticated {
						a.Patron = uiMsg.Username
					}
					a.ToUI <- bRes
				case "CHECKIN":
					a.State = uiCHECKIN
					a.ToRFID <- []byte(`{"Reader": "A", "Cmd": "SET-READER", "Data": "ON"}` + "\n")
				case "CHECKOUT":
					a.State = uiCHECKOUT
					a.ToRFID <- []byte(`{"Reader": "A", "Cmd": "SET-READER", "Data": "ON"}` + "\n")
				case "STATUS":
					a.State = uiSTATUS
				case "LOGOUT":
					a.State = uiWAITING
					a.Authenticated = false
					a.Patron = ""
					a.ToUI <- []byte(`{"action": "LOGOUT", "status": true}` + "\n")
					a.ToRFID <- []byte(`{"Reader": "A", "Cmd": "SET-READER", "Data": "OFF"}` + "\n")
				}
			}
		case <-a.Quit:
			// cleanup: close channels & connections
			close(a.ToUI)
			close(a.ToRFID)
			close(a.FromRFID)
			log.Println("INFO", "shutting down state machine", a.IP)
			if a.SIPConn != nil {
				a.SIPConn.Close()
				a.SIPConn = nil
			}
			return
		}
	}
}

// read from tcp connection and pipe into FromRFID channel
func (a *Automat) tcpReader() {
	r := bufio.NewReader(a.RFIDconn)
	for {
		msg, err := r.ReadBytes('\n')
		if err != nil {
			a.Quit <- true
			break
		}
		a.FromRFID <- msg
	}
}

// write messages from channel ToRFID to tcp connection
func (a *Automat) tcpWriter() {
	w := bufio.NewWriter(a.RFIDconn)
	for msg := range a.ToRFID {
		_, err := w.Write(msg)
		if err != nil {
			log.Println(err)
			break
		}
		log.Println("-> RFID:", strings.TrimRight(string(msg), "\n"))
		err = w.Flush()
		if err != nil {
			log.Println(err)
			break
		}
	}
}

// read from websocket connection and pipe into FromUI channel
func (a *Automat) wsReader() {
	for {
		// msgType, msg, err
		_, msg, err := a.ws.ReadMessage()
		if err != nil {
			break
		}
		a.FromUI <- msg
	}

}

// write messages from channel ToUI into websocket connection
func (a *Automat) wsWriter() {
	for msg := range a.ToUI {
		// TODO ws.WriteJSON() takes interface{}, i.e  go struct
		err := a.ws.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			break
		}
		log.Println("-> UI:", strings.TrimRight(string(msg), "\n"))
	}
}
