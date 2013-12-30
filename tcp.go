package main

import (
	"bufio"
	"log"
	"net"
	"strings"
	"time"
)

type TCPServer struct {
	listenAddr string
	// TOOD this map should use only IP as key, but use ip+port for now
	// so integration test is easy
	connections map[string]*TCPClient
	addChan     chan *TCPClient
	rmChan      chan *TCPClient
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
		connections: make(map[string]*TCPClient, 0),
		listenAddr:  ":" + cfg.TCPPort,
		addChan:     make(chan *TCPClient),
		rmChan:      make(chan *TCPClient),
	}
}

func (srv TCPServer) handleMessages() {
	ticker := time.NewTicker(time.Second * 60)
	for {
		select {
		case <-ticker.C:
			log.Println("TCP number of connections:", len(srv.connections))
		case client := <-srv.addChan:
			log.Printf("TCP [%v] client connected\n", client.conn.RemoteAddr())
			srv.connections[client.conn.RemoteAddr().String()] = client
			client.outgoing <- "Welcome!\n"
			stats.ClientsConnected.Inc(1)
		case client := <-srv.rmChan:
			log.Printf("TCP [%v] client disconnected\n", client.conn.RemoteAddr())
			delete(srv.connections, client.conn.RemoteAddr().String())
			stats.ClientsConnected.Dec(1)
		}
	}
}

func (srv TCPServer) handleConnection(c net.Conn) {
	client := newTCPClient(c)
	defer c.Close()

	// register client
	srv.addChan <- client

	// unregister when client.Read() returns
	defer func() {
		srv.rmChan <- client
	}()

	go client.Write()
	client.Read()
}

type TCPClient struct {
	conn     net.Conn
	outgoing chan string
	reader   *bufio.Reader
	writer   *bufio.Writer
}

func newTCPClient(c net.Conn) *TCPClient {
	client := &TCPClient{
		conn:     c,
		outgoing: make(chan string),
		reader:   bufio.NewReader(c),
		writer:   bufio.NewWriter(c),
	}
	return client
}

func (client *TCPClient) Read() {
	for {
		msg, err := client.reader.ReadBytes('\n')
		if err != nil {
			return
		}
		// parse msg
		log.Printf("<-- [%v] %v\n", client.conn.RemoteAddr(), strings.TrimRight(string(msg), "\r\n"))
	}
}

func (client *TCPClient) Write() {
	for msg := range client.outgoing {
		_, err := client.writer.WriteString(msg)
		if err != nil {
			log.Println(err)
			return
		}
		log.Printf("--> [%v] %v\n", client.conn.RemoteAddr(), strings.TrimRight(msg, "\r\n"))
		err = client.writer.Flush()
		if err != nil {
			log.Println(err)
			return
		}
	}
}
