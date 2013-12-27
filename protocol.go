package main

import "encoding/json"

// request message to RFID-reader
type request struct {
	Command  string
	Data     string
	ReaderID string
	TagID    string
	Barcode  string
}

// reponse message from RFID-reader
type response struct {
	Command  string
	Status   string
	ReaderID string
	TagID    string
	Barcode  string
}

func parseRequest(b []byte) (*request, error) {
	var req *request
	err := json.Unmarshal(b, req)
	if err != nil {
		return req, err
	}
	return req, nil
}
