package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
)

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

func statusHandler(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(stats.Export())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		return
	}
	c := &monitorConn{send: make(chan *exportMetrics), ws: ws}
	hub.mReg <- c
	defer func() {
		hub.mUnReg <- c
	}()
	go c.writer()
	c.reader()
}

func serveFile(filename string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filename)
	}
}
