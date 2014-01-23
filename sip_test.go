package main

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	"github.com/knakk/specs"
)

// fakeTCPConn is a mock of the net.Conn interface
type fakeTCPConn struct {
	buffer bytes.Buffer
	io.ReadWriter
}

func (c fakeTCPConn) Close() error                       { return nil }
func (c fakeTCPConn) LocalAddr() net.Addr                { return nil }
func (c fakeTCPConn) RemoteAddr() net.Addr               { return nil }
func (c fakeTCPConn) SetDeadline(t time.Time) error      { return nil }
func (c fakeTCPConn) SetReadDeadline(t time.Time) error  { return nil }
func (c fakeTCPConn) SetWriteDeadline(t time.Time) error { return nil }

// fakeAutomat creates a fake Automat with a mocked connection with teturns
// the suplied sipResponse when read from.
func fakeAutomat(sipResponse string) *Automat {
	var c fakeTCPConn
	bufferWriter := bufio.NewWriter(&c.buffer)
	c.ReadWriter = bufio.NewReadWriter(
		bufio.NewReader(bytes.NewBufferString(sipResponse)),
		bufferWriter)
	return &Automat{SIPConn: c}
}

func TestFieldPairs(t *testing.T) {
	s := specs.New(t)

	fields := pairFieldIDandValue("AOHUTL|AA2|AEFillip Wahl|BLY|CQY|CC5|PCPT|PIY|AFGreetings from Koha. |\r")
	tests := []specs.Spec{
		{9, len(fields)},
		{"HUTL", fields["AO"]},
		{"2", fields["AA"]},
		{"Fillip Wahl", fields["AE"]},
		{"Y", fields["BL"]},
		{"Y", fields["CQ"]},
		{"5", fields["CC"]},
		{"PT", fields["PC"]},
		{"Y", fields["PI"]},
		{"Greetings from Koha. ", fields["AF"]},
	}
	s.ExpectAll(tests)
}

func TestSIPPatronAuthentication(t *testing.T) {
	specs := specs.New(t)
	a := fakeAutomat("64              01220140123    093212000000030003000000000000AOHUTL|AA2|AEFillip Wahl|BLY|CQY|CC5|PCPT|PIY|AFGreetings from Koha. |\r")
	res, err := DoSIPCall(a, sipFormMsgAuthenticate("HUTL", "2", "pass"), authParse)

	specs.ExpectNil(err)
	specs.Expect(true, res.Authenticated)

}
