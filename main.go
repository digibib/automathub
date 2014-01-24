package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
)

// Application state //////////////////////////////////////////////////////////

var (
	hub       *wsHub
	cfg       *config
	stats     *appMetrics
	server    *TCPServer
	logFile   *os.File
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

	if cfg.LogToFile {
		logFile, err := os.OpenFile(cfg.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatal(err)
		}
		log.SetOutput(logFile)
	}

}

// Application entry point ////////////////////////////////////////////////////

func main() {
	if cfg.LogToFile {
		defer logFile.Close()
	}
	// TCP server handles the communcation with the RFID-service on the
	// self-checkin-automats, and spins up an automat state-machine for every
	// connection.
	server = newTCPServer(cfg)
	go server.run()

	// Websocket server handles feedback to the user interface on self-checkin-
	// automats, and broadcast metrics to a monitor page.
	go hub.run()

	// HTTP handlers
	http.HandleFunc("/css/styles.css", serveFile("data/css/styles.css"))
	http.HandleFunc("/.status", statusHandler)
	http.HandleFunc("/js/JSXTransformer-0.8.0.js", serveFile("data/js/JSXTransformer-0.8.0.js"))
	http.HandleFunc("/js/react-with-addons-0.8.0.js", serveFile("data/js/react-with-addons-0.8.0.js"))
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/ui", uiHandler)
	http.HandleFunc("/", monitorHandler)

	// HTTP Server
	err := http.ListenAndServe(":"+cfg.HTTPPort, nil)
	if err != nil {
		log.Fatal(err)
	}
}
