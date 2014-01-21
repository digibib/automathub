package main

import "encoding/json"

// Automathub <-> RFIDService ////////////////////////////////////////////////

// request message to RFID-reader
type RFIDRequest struct {
	Command  string
	Data     string
	ReaderID string
	TagID    string
	Barcode  string
}

// reponse message from RFID-reader
type RFIDResponse struct {
	Command  string
	Status   string
	ReaderID string
	TagID    string
	Barcode  string
}

func parseRFIDResponse(b []byte) (*RFIDresponse, error) {
	var resp *RFIDresponse
	err := json.Unmarshal(b, resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func parseRFIDRequest(b []byte) (*RFIDrequest, error) {
	var req *RFIDrequest
	err := json.Unmarshal(b, req)
	if err != nil {
		return req, err
	}
	return req, nil
}

// Automat state machine <-> User interface ///////////////////////////////////

// request from UI to the state machine
type UIRequest struct {
	Action   string
	Username string
	PIN      string
}

// response from the state machine to UI
type UIRespoinse struct {
	Action  string
	Status  string
	Message string
}
