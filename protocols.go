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

func parseRFIDResponse(b []byte) (RFIDResponse, error) {
	var resp RFIDResponse
	err := json.Unmarshal(b, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func parseRFIDRequest(b []byte) (RFIDRequest, error) {
	var req RFIDRequest
	err := json.Unmarshal(b, &req)
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
type UIResponse struct {
	Action        string
	Status        string
	Patron        string
	Authenticated bool
	Message       string
	ErrorDetails  string
	Item          item
	// Loans         []item
	// Holdings      []item
}

type item struct {
	Title  string // [bok] Forfatter - tittel
	Status string // forfaller 10/03/2013
	OK     bool   // false = mangler brikke / klarte ikke lese den
}

func ErrorResponse(errMsg error) []byte {
	b, err := json.Marshal(&UIResponse{
		Action:       "ERROR",
		Message:      "Noe gikk galt, det er ikke din feil!",
		ErrorDetails: errMsg.Error()})
	if err != nil {
		return []byte(`{"Action": "ERROR", "Message": "something went wrong, not your fault!", "ErrorDetails": "kuk i kompjuderen"}`)
	}
	return b
}
