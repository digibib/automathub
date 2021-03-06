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
	sipMsg93 = "9300CNstresstest%d|COstresstest%d|CPHUTL|\r"

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

// DoSIPCall performs a SIP request with an automat's SIP TCP-connection. It
// takes a SIP message as a string and a parser function to transform the SIP
// response into a UIResponse.
func DoSIPCall(p *ConnPool, req string, parser parserFunc) (*UIResponse, error) {
	// 0. Get connection from pool
	c := p.Get()
	defer p.Release(c)

	// 1. Send the SIP request
	_, err := c.Write([]byte(req))
	if err != nil {
		return nil, err
	}

	log.Println("-> SIP", strings.Trim(req, "\n\r"))

	// 2. Read SIP response
	reader := bufio.NewReader(c)
	resp, err := reader.ReadString('\r')
	if err != nil {
		return nil, err
	}

	log.Println("<- SIP", strings.Trim(resp, "\n\r"))

	// 3. Parse the response
	res := parser(resp)

	return res, nil
}

func authParse(s string) *UIResponse {
	b := s[61:] // first part of SIPresponse not needed here
	fields := pairFieldIDandValue(b)

	var auth bool
	if fields["CQ"] == "Y" {
		auth = true
	}
	return &UIResponse{Action: "LOGIN", Authenticated: auth, Patron: fields["AA"]}
}

func checkinParse(s string) *UIResponse {
	a, b := s[:24], s[24:]
	var (
		ok     bool
		status string
	)
	if a[2] == '1' {
		ok = true
	}
	fields := pairFieldIDandValue(b)
	if a[2] == '0' {
		status = fields["AF"]
	} else {
		status = fmt.Sprintf("registrert innlevert %s/%s/%s", a[12:14], a[10:12], a[6:10])
	}
	return &UIResponse{Item: item{OK: ok, Title: fields["AJ"], Status: status}}
}

func checkoutParse(s string) *UIResponse {
	a, b := s[:24], s[24:]
	var (
		ok     bool
		status string
	)
	fields := pairFieldIDandValue(b)
	if a[2] == '1' {
		ok = true
		date := fields["AH"]
		status = fmt.Sprintf("utlånt til %s/%s/%s", date[6:8], date[4:6], date[0:4])
	} else {
		if fields["AF"] == "1" {
			status = "Failed! Don't know why; SIP should give more information"
		} else {
			status = fields["AF"]
		}
	}
	return &UIResponse{Item: item{OK: ok, Status: status, Title: fields["AJ"]}}
}
