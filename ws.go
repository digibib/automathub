package main

import (
	"log"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

type uiConn struct {
	ws       *websocket.Conn
	clientIP net.Addr
	send     chan []byte
}

type monitorConn struct {
	ws   *websocket.Conn
	send chan *exportMetrics
}

func (c *monitorConn) writer() {
	for message := range c.send {
		err := c.ws.WriteJSON(message)
		if err != nil {
			break
		}
	}
}

func (c *monitorConn) reader() {
	for {
		_, _, err := c.ws.ReadMessage()
		if err != nil {
			break
		}
	}
}

type wsHub struct {
	monitor *monitorConn
	mReg    chan *monitorConn
	mUnReg  chan *monitorConn
	// broadcast chan []byte
	// register   chan uiConn
	// unregister chan uiConn
}

func (h *wsHub) run() {
	ticker := time.NewTicker(time.Second * 1)
	for {
		select {
		case <-ticker.C:
			if h.monitor != nil {
				h.monitor.send <- stats.Export()
			}
		case c := <-h.mReg:
			log.Println("WS  Monitor connected")
			h.monitor = c
		case <-h.mUnReg:
			close(h.monitor.send)
			go h.monitor.ws.Close()
			h.monitor = nil
			log.Println("WS  Monitor disconnected")
		}
	}
}
