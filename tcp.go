package main

import (
	"log"
	"net"
	"time"
)

type TCPServer struct {
	listenAddr string
	// TODO this map should use only IP as key, but use ip+port for now
	// so integration test is easy on localhost (=same ip for all connections)
	connections map[string]*Automat
	addChan     chan *Automat
	rmChan      chan *Automat
}

func (srv TCPServer) run() {
	ln, err := net.Listen("tcp", srv.listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	go srv.handleMessages()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go srv.handleConnection(conn)
	}
}

func newTCPServer(cfg *config) *TCPServer {
	return &TCPServer{
		connections: make(map[string]*Automat, 0),
		listenAddr:  ":" + cfg.TCPPort,
		addChan:     make(chan *Automat),
		rmChan:      make(chan *Automat),
	}
}

func (srv TCPServer) get(addr string) <-chan *Automat {
	c := make(chan *Automat)
	for {
		go func() {
			if a, ok := srv.connections[addr]; ok {
				c <- a
				return
			}
		}()
		return c
	}
}

func (srv TCPServer) handleMessages() {
	ticker := time.NewTicker(time.Second * 60)
	for {
		select {
		case <-ticker.C:
			//log.Println("TCP number of connections:", len(srv.connections))
		case automat := <-srv.addChan:
			log.Printf("TCP [%v] automat connected\n", automat.RFIDconn.RemoteAddr())
			srv.connections[automat.RFIDconn.RemoteAddr().String()] = automat
			stats.ClientsConnected.Inc(1)
		case automat := <-srv.rmChan:
			log.Printf("TCP [%v] automat disconnected\n", automat.RFIDconn.RemoteAddr())
			// close ws connection
			if automat.ws != nil { // panics if no connection
				automat.ws.Close()
			}
			delete(srv.connections, automat.RFIDconn.RemoteAddr().String())
			stats.ClientsConnected.Dec(1)
		}
	}
}

func (srv TCPServer) handleConnection(c net.Conn) {
	automat := newAutomat(c)
	defer c.Close()

	// register automat
	srv.addChan <- automat

	// unregister when automat.Read() returns
	defer func() {
		srv.rmChan <- automat
	}()

	go automat.run()
	go automat.tcpWriter()
	automat.tcpReader()
}
