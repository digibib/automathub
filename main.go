package main

import (
	"html/template"
	"log"
	"net/http"
)

// Application state //////////////////////////////////////////////////////////

var (
	hub       *wsHub
	cfg       *config
	stats     *appMetrics
	server    *TCPServer
	templates = template.Must(
		template.ParseFiles("data/html/monitor.html", "data/html/ui.html"))
)

// Setup //////////////////////////////////////////////////////////////////////

func init() {
	cfg = &config{}
	err := cfg.fromFile("config.json")
	if err != nil {
		log.Fatal(err)
	}

	stats = RegisterMetrics()

	hub = NewHub()
}

// Application entry point ////////////////////////////////////////////////////

func main() {
	// TCP server handles the communcation with the RFID-service on the
	// self-checkin-automats, dispatches to SIP-server and handles all the
	// business logic.
	server = newTCPServer(cfg)
	go server.run()

	// Websocket server handles feedback to the user interface on self-checkin-
	// automats, and broadcast metrics to a monitor page.
	go hub.run()

	// HTTP handlers
	http.HandleFunc("/css/styles.css", serveFile("data/css/styles.css"))
	http.HandleFunc("/.status", statusHandler)
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/ui", uiHandler)
	http.HandleFunc("/", monitorHandler)

	// HTTP Server
	err := http.ListenAndServe(":"+cfg.HTTPPort, nil)
	if err != nil {
		log.Fatal(err)
	}
}
