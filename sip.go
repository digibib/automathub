package main

import (
	"fmt"
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
