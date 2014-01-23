package main

import (
	"bufio"
	"fmt"
	"log"
	"strings"
	"time"
)

const (
	// Transaction date format
	sipDateLayout = "20060102    150405"

	// 93: Login (established SIP connection)
	sipMsg93 = "9300CNstresstest1|COstresstest1|CPHUTL|\r"

	// 63: Patron information request
	sipMsg63 = "63012%v          AO%s|AA%s|AC<terminalpassword>|AD%s|BP000|BQ9999|\r"

	// 09: Chekin
	sipMsg09 = "09N%v%vAP<location>|AO%v|AB%v|AC<terminalpassword>|\r"

	// 11: Checkout
	sipMsg11 = "11YN%v%vAO<institutionid>|AA%s|AB%s|AC<terminalpassword>|\r"
)

// TODO investigate SIP fileds, do Koha need them to be filled out?:
// <terminalpassword>
// <location>
// <institutionid>

func sipFormMsgAuthenticate(dept, username, pin string) string {
	now := time.Now().Format(sipDateLayout)
	return fmt.Sprintf(sipMsg63, now, dept, username, pin)
}

func sipFormMsgCheckin(dept, barcode string) string {
	now := time.Now().Format(sipDateLayout)
	return fmt.Sprintf(sipMsg09, now, now, dept, barcode)
}

func sipFormMsgCheckout(username, barcode string) string {
	now := time.Now().Format(sipDateLayout)
	return fmt.Sprintf(sipMsg11, now, now, username, barcode)
}

func pairFieldIDandValue(msg string) map[string]string {
	results := make(map[string]string)

	for _, pair := range strings.Split(strings.TrimRight(msg, "|\r"), "|") {
		id, val := pair[0:2], pair[2:]
		results[id] = val
	}
	return results
}

// A parserFunc parses a SIP response. It extracts the desired information and
// returns the JSON message to be sent to the user interface.
type parserFunc func(string) *UIResponse

// DoSIPCall performs a SIP request with the a's SIP TCP-connection. It takes
// a SIP message as a string and a parser function to transform the SIP
// response into a JSON message
func DoSIPCall(a *Automat, req string, parser parserFunc) (*UIResponse, error) {
	// 1. Ensure we have a TCP SIP connection

	// TODO conn peek?

	// 2. Send the SIP request
	_, err := a.SIPConn.Write([]byte(req))
	if err != nil {
		return nil, err
	}

	log.Println("-> SIP", strings.Trim(req, "\n\r"))

	// 3. Read SIP response
	reader := bufio.NewReader(a.SIPConn)
	resp, err := reader.ReadString('\r')
	if err != nil {
		return nil, err
	}

	log.Println("<- SIP", strings.Trim(resp, "\n\r"))

	// 4. Parse the response
	res := parser(resp)

	return res, nil
}

func authParse(s string) *UIResponse {
	_, b := s[:61], s[61:] // TODO use the first part of sipmessage
	fields := pairFieldIDandValue(b)

	var auth bool
	if fields["CQ"] == "Y" {
		auth = true
	}
	return &UIResponse{Action: "LOGIN", Authenticated: auth}
}
