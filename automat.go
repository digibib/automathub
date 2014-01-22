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
	uiLOGGEDIN
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

	// SIP connection (via TCP)
	// This is closed when as long as State is uiWAITING, otherwise keep alive
	// to reuse TCP connection.
	SIPConn   net.Conn
	SIPReader *bufio.Reader

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

// connect to SIP server
func (a *Automat) ensureSIPConnection() {
	if a.SIPConn != nil {
		return
	}
	conn, err := net.Dial("tcp", cfg.SIPServer)
	if err != nil {
		// TODO return & handle connection error
		println(err)
		return
	}
	a.SIPConn = conn
	a.SIPReader = bufio.NewReader(a.SIPConn)
	// send sip login message
	_, err = a.SIPConn.Write([]byte(sipMsg93))
	if err != nil {
		// TODO handle this
		println(err)
	}
	log.Println("-> SIP", strings.Trim(sipMsg93, "\n\r"))
	msg, err := a.SIPReader.ReadString('\r')
	if err != nil {
		// TODO handle this
		println(err)
	}
	log.Println("<- SIP", strings.Trim(msg, "\n\r"))
}

// disconnect from SIP server
func (a *Automat) closeSIPConnection() {
	a.SIPConn.Close()
	a.SIPConn = nil
	a.SIPReader = nil
}

// run the Automat state machine & message handler
func (a *Automat) run() {
	for {
		select {
		case msg := <-a.FromRFID:
			a.ToUI <- msg
			log.Println("<- RFID:", strings.TrimRight(string(msg), "\n"))
		case msg := <-a.FromUI:
			log.Println("<- UI", strings.TrimRight(string(msg), "\n"))
			var uiMsg UIRequest
			err := json.Unmarshal(msg, &uiMsg)
			if err != nil {
				a.ToUI <- []byte(`{"error": "something went wrong, not your fault!"}`)
			} else {
				switch uiMsg.Action {
				case "LOGIN":
					a.ensureSIPConnection()
					authenticated := PatronAuthenticate(a, uiMsg.Username, uiMsg.PIN)
					if authenticated {
						a.Authenticated = true
						a.ToUI <- []byte(`{"action": "LOGIN", "status": true}`)
					} else {
						a.Authenticated = false
						a.ToUI <- []byte(`{"action": "LOGIN", "status": false}`)
					}
				case "LOGOUT":
					a.State = uiWAITING
					a.Authenticated = false
					a.closeSIPConnection()
					a.ToUI <- []byte(`{"action": "LOGOUT", "status": true}`)
				}
			}
		case <-a.Quit:
			// cleanup: close channels & connections
			close(a.ToUI)
			close(a.ToRFID)
			log.Println("shutting down state machine", a.IP)
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
