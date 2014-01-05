package main

import (
	"bufio"
	"log"
	"net"

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
			println("-> RFID:", msg)
		case msg := <-a.ToRFID:
			println("<- RFID:", msg)
		case msg := <-a.FromUI:
			println("<- UI", msg)
		case msg := <-a.ToUI:
			println("-> UI:", msg)
		case <-a.Quit:
			// cleanup: close channels & connections
			println("shutting down state machine")
		}
	}
}

// read from tcp connection and pipe into FromRFID channel
func (a *Automat) tcpRead() {
	r := bufio.NewReader(a.conn)
	for {
		msg, err := r.ReadBytes('\n')
		if err != nil {
			return
		}
		a.FromRFID <- msg
	}
}

// write messages from channel ToRFID to tcp connection
func (a *Automat) tcpWrite() {
	w := bufio.NewWriter(a.conn)
	for msg := range a.ToRFID {
		_, err := w.Write(msg)
		if err != nil {
			log.Println(err)
			return
		}
		err = w.Flush()
		if err != nil {
			log.Println(err)
			return
		}
	}
}

// read from websocket connection and pipe into FromUI channel
func (a *Automat) wsRead() {
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
func (a *Automat) wsWrite() {
	for msg := range a.ToUI {
		err := a.ws.WriteJSON(msg)
		if err != nil {
			break
		}
	}
}
