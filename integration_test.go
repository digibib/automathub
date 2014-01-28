// +build integration

package main

import (
	"bufio"
	"fmt"
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
	NUMCLIENTS = 50

	// How long to run the integration test
	DURATION = 20 * time.Second

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
		case <-s.incoming:
			//log.Println("incoming", string(msg))
		}

	}
}

func login(p *patron, a *Automat) {
	msg := fmt.Sprintf(`{"Action": "LOGIN", "Username":"%s", "Pin": "pass"}`, p.ID)
	a.FromUI <- []byte(msg)
}

func logout(p *patron, a *Automat) {
	msg := fmt.Sprintf(`{"Action": "LOGOUT", "Username":"%s"}`, p.ID)
	a.FromUI <- []byte(msg)
}

func checkin(p *patron, a *Automat) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	if len(p.Checkouts) == 0 {
		//println("     patron has no checkouts")
		return
	}
	var item string
	for i := 0; i < rand.Intn(len(p.Checkouts)); i++ {
		item, p.Checkouts = p.Checkouts[len(p.Checkouts)-1],
			p.Checkouts[:len(p.Checkouts)-1]
		msg := fmt.Sprintf(`{"Barcode": "%s"}`, item)
		a.FromRFID <- []byte(msg)
		println("     checkin item:", item)
	}
}

func checkout(p *patron, a *Automat) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	for i := 0; i < rand.Intn(MAXCHEKCOUTS); i++ {
		item := items[rand.Intn(len(items))]
		p.Checkouts = append(p.Checkouts, item)
		//println("     checkout item:", item)
		msg := fmt.Sprintf(`{"Barcode": "%s"}`, item)
		a.FromRFID <- []byte(msg)
	}
}

func simulatePatronAutomatInteraction() {
	s := newRFIDService()
	err := s.connect()
	if err != nil {
		return
	}

	go s.handleMessages()
	a := <-server.get(s.conn.LocalAddr().String())
	go func() {
		for _ = range a.ToUI {
			// discarding
			// log.Printf(string(msg))
			//println("simulating send to UI")
		}
	}()

	for {
		action := rand.Intn(100)
		switch {
		case action < 40 && action > 0:
			patron := patrons[rand.Intn(len(patrons))]
			login(patron, a)
			a.FromUI <- []byte(`{"Action":"CHECKIN"}`)
			checkin(patron, a)
			logout(patron, a)
		case action > 40 && action < 80:
			patron := patrons[rand.Intn(len(patrons))]
			login(patron, a)
			a.FromUI <- []byte(`{"Action":"CHECKOUT"}`)
			checkout(patron, a)
			logout(patron, a)
		case action > 80:
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(10000)+1))
		}
	}
}

// SETUP

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

// TESTS

func TestAutomatPatronInteraction(t *testing.T) {
	s := specs.New(t)
	server = newTCPServer(&config{TCPPort: "6666"})
	rand.Seed(time.Now().UnixNano())
	if sipPool.size == 0 {
		log.Fatal("No SIP connections")
	}
	go server.run()
	s.Expect(0, len(server.connections))

	for i := 0; i < NUMCLIENTS; i++ {
		go simulatePatronAutomatInteraction()

	}

	time.Sleep(DURATION)
	s.Expect(NUMCLIENTS, len(server.connections))

	// TODO iterate over patrons and checkin all checked out books
	for i := range patrons {
		for _, j := range patrons[i].Checkouts {
			_, _ = DoSIPCall(sipPool, sipFormMsgCheckin("HUTL", j), checkinParse)
			println(i, j)
		}
	}
}
