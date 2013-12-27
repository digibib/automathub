package main

import (
	"encoding/json"
	"io/ioutil"
)

type automat struct {
	IP   string
	Name string
}

type config struct {
	TCPPort  string
	HTTPPort string
	Automats []automat
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
