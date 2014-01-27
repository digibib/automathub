package main

import (
	"encoding/json"
	"io/ioutil"
)

type automat struct {
	IP         string
	Name       string
	Department string
}

type config struct {
	LogFile           string
	LogToFile         bool
	NumSIPConnections int
	SIPServer         string
	TCPServer         string
	TCPPort           string
	HTTPPort          string
	Automats          []automat
}

func (c *config) fromFile(file string) error {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, c)
	if err != nil {
		return err
	}
	return nil
}
