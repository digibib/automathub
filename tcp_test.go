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

const NUMCLIENTS = 100

func connect() {
	conn, err := net.Dial("tcp", "localhost:6666")
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	r := bufio.NewReader(conn)
	for {
		_, err := r.ReadString('\n')
		if err != nil {
			break
		}
	}
}

func TestTCPServer(t *testing.T) {
	s := specs.New(t)
	server := newTCPServer(&config{TCPPort: "6666"})

	go server.run()
	s.Expect(0, len(server.connections))

	for i := 0; i < NUMCLIENTS; i++ {
		go connect()
	}
	time.Sleep(time.Second)
	s.Expect(NUMCLIENTS, len(server.connections))
}
