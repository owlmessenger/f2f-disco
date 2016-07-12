package frontend

import (
	"time"
)

type Backend interface {
	Lookup([]byte) []Addr
	Announce(Entry)
	DeleteAfter(time.Time)
}

type Entry struct {
	Addr      Addr
	ID        []byte
	Timestamp time.Time
}
