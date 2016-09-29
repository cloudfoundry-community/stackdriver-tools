package dev

import "fmt"

type StdOut struct {
}

func (so *StdOut) Connect() bool {
	return true
}

func (so *StdOut) ShipEvents(event map[string]interface{}, msg string) {
	fmt.Printf("%s: %+v\n\n", msg, event)
}

