package main

import (
	"bufio"
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

// Automat is a state machine for the automats. It handles all communications
// with the RFID-service and the User interface.
type Automat struct {
	State uiState
	IP    string // remote address of the automat

	// Communication with the RFID service (via TCP)
	conn     net.Conn
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
		conn:     c,
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
			a.ToUI <- msg
			log.Println("<- RFID:", strings.TrimRight(string(msg), "\n"))
		case msg := <-a.FromUI:
			log.Println("<- UI", strings.TrimRight(string(msg), "\n"))
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
	r := bufio.NewReader(a.conn)
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
	w := bufio.NewWriter(a.conn)
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