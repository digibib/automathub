package main

import (
	"testing"

	"github.com/knakk/specs"
)

func TestMsgAuthenticate(t *testing.T) {
	specs := specs.New(t)

	msg := sipFormMsgAuthenticate("HUTL", "N0012341234", "9999")
	specs.ExpectMatches(`^63012`, msg)
	specs.ExpectMatches(`AOHUTL\|AAN0012341234\|AC<terminalpassword>\|AD9999\|BP000\|BQ9999\|\r$`, msg)
}

func TestMsgCheckin(t *testing.T) {
	specs := specs.New(t)

	msg := sipFormMsgCheckin("HUTL", "0301234125789")
	specs.ExpectMatches(`^09N`, msg)
	specs.ExpectMatches(`AP<location>\|AOHUTL\|AB0301234125789\|AC<terminalpassword>\|\r$`, msg)
}

func TestMsgCheckout(t *testing.T) {
	specs := specs.New(t)

	msg := sipFormMsgCheckout("N001234", "0301234125789")
	specs.ExpectMatches(`^11YN`, msg)
	specs.ExpectMatches(`AO<institutionid>\|AAN001234\|AB0301234125789\|AC<terminalpassword>\|\r$`, msg)
}
