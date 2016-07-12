package frontend

import (
	"golang.org/x/crypto/ed25519"
	"time"
)

const (
	MessageAnnounce = iota
	MessageLookup
)

type Message struct {
	Type uint8

	// Annouce and Request
	ID ed25519.PublicKey

	// Announce
	Timestamp time.Time
	Sig       []byte
}

func (m *Message) Verify() bool {
	publicKey := m.ID
	message, _ := m.Timestamp.MarshalBinary()
	signature := m.Sig
	return ed25519.Verify(publicKey, message, signature)
}

type Reply struct {
	ID    ed25519.PublicKey
	Addrs []Addr
}

type Addr struct {
	Protocol string
	Host     string
}
