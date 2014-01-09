// +build integration

package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/knakk/specs"
)

const (
	// Number of simulated automats with RFIDService
	NUMCLIENTS = 100

	// How long to run the integration test
	DURATION = 200 * time.Second

	// TCPServer addresss
	HOST = "localhost:6666"

	// File containing barcodes; one per line
	ITEMSFILE = "testdata/items.txt"

	// File containing patron ids; one per line. Assumed password: "pass".
	PATRONSFILE = "testdata/patrons.txt"

	// Maximum patron item checkouts per session
	MAXCHEKCOUTS = 20
)

type RFIDState uint

const (
	RFIDWaiting RFIDState = iota
	RFIDReading
)

type RFIDService struct {
	state    RFIDState
	conn     net.Conn
	incoming chan []byte
	outgoing chan []byte
}

type patron struct {
	sync.Mutex
	ID        string
	Checkouts []string
}

var (
	patrons []*patron
	items   []string
)

func newRFIDService() *RFIDService {
	return &RFIDService{
		state:    RFIDWaiting,
		incoming: make(chan []byte),
		outgoing: make(chan []byte)}
}

func (s *RFIDService) connect() error {
	conn, err := net.Dial("tcp", HOST)
	if err != nil {
		return err
	}
	s.conn = conn
	return nil
}

func (s *RFIDService) reader() {
	r := bufio.NewReader(s.conn)
	for {
		msg, err := r.ReadBytes('\n')
		if err != nil {
			close(s.outgoing)
			return
		}
		s.incoming <- msg
	}
}

func (s *RFIDService) writer() {
	w := bufio.NewWriter(s.conn)
	for msg := range s.outgoing {
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

func (s *RFIDService) handleMessages() {
	defer s.conn.Close()

	go s.reader()
	go s.writer()
	for {
		select {
		case msg := <-s.incoming:
			log.Println("incoming", string(msg))
		}

	}
}

func init() {
	// load & init patrons
	f1, err := os.Open(PATRONSFILE)
	if err != nil {
		log.Fatal(err)
	}
	defer f1.Close()
	scanner := bufio.NewScanner(f1)
	for scanner.Scan() {
		patrons = append(patrons, &patron{ID: scanner.Text()})
	}

	// load items
	f2, err := os.Open(ITEMSFILE)
	if err != nil {
		log.Fatal(err)
	}
	defer f2.Close()
	reader2 := bufio.NewReader(f2)
	contents, err := ioutil.ReadAll(reader2)
	if err != nil {
		log.Fatal(err)
	}
	items = strings.Split(string(contents), "\n")
}

func checkin(p *patron) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	if len(p.Checkouts) == 0 {
		log.Println("     patron has no checkouts")
		return
	}
	var item string
	for i := 0; i < rand.Intn(len(p.Checkouts)); i++ {
		item, p.Checkouts = p.Checkouts[len(p.Checkouts)-1],
			p.Checkouts[:len(p.Checkouts)-1]
		log.Printf("     checkin item: %v\n", item)
	}
}

func checkout(p *patron) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	for i := 0; i < rand.Intn(MAXCHEKCOUTS); i++ {
		item := items[rand.Intn(len(items))]
		p.Checkouts = append(p.Checkouts, item)
		log.Printf("     checkout item: %v\n", item)
	}
}

func simulateAutomat() {
	s := newRFIDService()
	err := s.connect()
	if err != nil {
		return
	}

	go s.handleMessages()
	a := s.conn.LocalAddr().String()

	for {
		action := rand.Intn(100)
		switch {
		case action < 40 && action > 0:
			patron := patrons[rand.Intn(len(patrons))]
			log.Printf("[%v] simulating checkin patron: %v\n", a, patron.ID)
			checkin(patron)
		case action > 40 && action < 80:
			patron := patrons[rand.Intn(len(patrons))]
			log.Printf("[%v] simulating login patron: %v\n", a, patron.ID)
			checkout(patron)
		case action > 80:
			log.Printf("[%v] simulating wait\n", a)
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)+1))
		}
	}
}

func TestAutomatPatronInteraction(t *testing.T) {
	s := specs.New(t)
	server := newTCPServer(&config{TCPPort: "6666"})
	rand.Seed(time.Now().UnixNano())

	go server.run()
	s.Expect(0, len(server.connections))

	for i := 0; i < NUMCLIENTS; i++ {
		go simulateAutomat()

	}

	time.Sleep(DURATION)
	s.Expect(NUMCLIENTS, len(server.connections))
}
