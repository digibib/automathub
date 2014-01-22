package main

import (
	"fmt"
	"log"
	"regexp"
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

func PatronAuthenticate(a *Automat, username, pin string) bool {
	sipUt := sipFormMsgAuthenticate(a.Dept, username, pin)
	_, err := a.SIPConn.Write([]byte(sipUt))
	if err != nil {
		println(err)
		return false
	}
	log.Println("-> SIP", strings.Trim(sipUt, "\n\r"))
	sipIn, err := a.SIPReader.ReadString('\r')
	if err != nil {
		println(err)
		return false
	}
	log.Println("<- SIP", strings.Trim(sipIn, "\n\r"))
	return rAuthenticated.MatchString(sipIn)
}
