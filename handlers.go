package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// monitorHandler serves the monitor pages
func monitorHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Host     string
		Automats []automat
	}{
		r.Host,
		cfg.Automats,
	}
	err := templates.ExecuteTemplate(w, "monitor.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// uiHandler serves the user interface of the automats
func uiHandler(w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query()
	if _, ok := server.connections[v.Get("client")]; !ok {
		http.Error(w, "ERROR: no automat connected with that address", http.StatusBadRequest)
		return
	}
	data := struct {
		Host      string
		Client    string
		JSXPragma template.JS
	}{
		r.Host,
		v.Get("client"),
		template.JS("/** @jsx React.DOM */"),
	}
	err := templates.ExecuteTemplate(w, "ui.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(stats.Export())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// wsHandler establishes connections with monitor pages and the automat-UIs
func wsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		return
	}

	v := r.URL.Query()
	if v.Get("client") == "monitor" {
		// Monitor connection
		c := &monitorConn{send: make(chan *exportMetrics), ws: ws}
		hub.mReg <- c
		defer func() {
			hub.mUnReg <- c
		}()
		go c.writer()
		c.reader()
	} else {
		// UI connection
		select {
		case a := <-server.get(v.Get("client")):
			a.ws = ws
			log.Println("UI", a.IP, "connected")

			defer func() {
				log.Println("UI", a.IP, "disconnected")
				//close(a.ToUI)
				go a.ws.Close()
			}()

			go a.wsWriter()
			a.ToUI <- []byte("{\"msg\":\"hei & velkommen\"}")
			a.wsReader()
			// TODO rethink
			// Instead og go.wswriter & a.wsreader
			// wait for ?
			// <- a.quit (quit: make(chan bool, 1))
		case <-time.After(time.Second * 3):
			return
		}
	}
}

// serveFile serves a single file from disk
func serveFile(filename string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filename)
	}
}
