package main

import (
	"bufio"
	"fmt"
	"net"
	"regexp"
	"time"
)

const (
	// Transaction date format
	sipDateLayout = "20060102    150405"

	// 63: Patron information request
	sipMsg63 = "63012%v          AO%s|AA%s|AC<terminalpassword>|AD%s|BP000|BQ9999|"

	// 09: Chekin
	sipMsg09 = "09N%v%vAP<location>|AO%v|AB%v|AC<terminalpassword>|"

	// 11: Checkout
	sipMsg11 = "11YN%v%vAO<institutionid>|AA%s|AB%s|AC<terminalpassword>|"
)

var (
	rAuthenticated = regexp.MustCompile(`\|CQY\|`)
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

func PatronAuthenticate(conn net.Conn, dept, username, pin string) bool {
	_, err := conn.Write([]byte(sipFormMsgAuthenticate(dept, username, pin)))
	if err != nil {
		return false
	}
	r := bufio.NewReader(conn)
	msg, err := r.ReadString('\r') // blocks! nothing coming from sip
	if err != nil {
		return false
	}
	return rAuthenticated.MatchString(msg)
}
