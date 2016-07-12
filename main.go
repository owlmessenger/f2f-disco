package main

import (
	"github.com/owlmessenger/f2f-disco/frontend"
	"github.com/owlmessenger/f2f-disco/inmembackend"
	"log"
)

const (
	UDPAddr = "8001"
	TCPAddr = ":8000"
)

func main() {
	backend := inmembackend.New()
	ds, err := frontend.NewDiscoServer(backend)
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(ds.Serve())
}
