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
	monitors map[*monitorConn]bool // Connected monitor pages
	mReg     chan *monitorConn     // Register monitor
	mUnReg   chan *monitorConn     // Unregister monitor
}

func NewHub() *wsHub {
	return &wsHub{
		monitors: make(map[*monitorConn]bool),
		mReg:     make(chan *monitorConn),
		mUnReg:   make(chan *monitorConn),
	}
}

func (h *wsHub) run() {
	ticker := time.NewTicker(time.Second * 1)
	for {
		select {
		case <-ticker.C:
			m := stats.Export()
			for c := range h.monitors {
				select {
				case c.send <- m:
				default:
					delete(h.monitors, c)
					close(c.send)
					go c.ws.Close()
				}
			}
		case c := <-h.mReg:
			h.monitors[c] = true
			log.Println("WS  Monitor connected")
		case c := <-h.mUnReg:
			delete(h.monitors, c)
			close(c.send)
			log.Println("WS  Monitor disconnected")

		}
	}
}
