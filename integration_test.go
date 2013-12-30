// +build integration

package main

import (
	"bufio"
	"log"
	"net"
	"testing"
	"time"

	"github.com/knakk/specs"
)

const (
	// Number of simulated automats with RFIDService
	NUMCLIENTS = 100
	// How long to run the integration test
	DURATION = 2 * time.Second
	// TCPServer addresss
	HOST = "localhost:6666"
	// File containing barcodes; one per line
	ITEMSFILE = "items.txt"
)

type RFIDService struct {
	// TCP connection
	conn net.Conn

	// communication channels
	incoming chan []byte
	outgoing chan []byte
	closing  chan bool
}

func NewRFIDService() *RFIDService {
	return &RFIDService{
		incoming: make(chan []byte),
		outgoing: make(chan []byte),
		closing:  make(chan bool),
	}
}

func (a *RFIDService) connect() error {
	conn, err := net.Dial("tcp", HOST)
	if err != nil {
		return err
	}
	a.conn = conn
	return nil
}

func (a *RFIDService) reader() {
	r := bufio.NewReader(a.conn)
	for {
		msg, err := r.ReadBytes('\n')
		if err != nil {
			a.closing <- true
		}
		a.incoming <- msg
	}
}

func (a *RFIDService) writer() {
	w := bufio.NewWriter(a.conn)
	for msg := range a.outgoing {
		_, err := w.Write(msg)
		if err != nil {
			log.Println(err)
		}
		err = w.Flush()
		if err != nil {
			log.Println(err)
		}
	}
}

func (a *RFIDService) run() {
	defer a.conn.Close()

	go a.reader()
	go a.writer()

	for {
		select {
		case msg := <-a.incoming:
			log.Println("incoming", string(msg))
		case <-a.closing:
			log.Println("connection closing")
			return
		}

	}

}

func TestTCPServer(t *testing.T) {
	s := specs.New(t)
	server := newTCPServer(&config{TCPPort: "6666"})

	go server.run()
	s.Expect(0, len(server.connections))

	for i := 0; i < NUMCLIENTS; i++ {
		c := NewRFIDService()
		c.connect()
		go c.run()
	}
	time.Sleep(DURATION)
	s.Expect(NUMCLIENTS, len(server.connections))
}
