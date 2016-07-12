package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/ed25519"
	"log"
	"net"
	"time"
)

const (
	UDPAddr = "8001"
	TCPAddr = ":8000"
)

type Entry struct {
	Protocol  string
	Addr      string
	ID        ed25519.PublicKey
	Timestamp time.Time
}

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

type DiscoServer struct {
	tcp net.Listener
	udp *net.UDPConn
	db  sql.DB

	announce   *sql.Stmt
	lookup     *sql.Stmt
	getExpired *sql.Stmt
}

func NewDiscoServer() (ds *DiscoServer, err error) {
	ds = &DiscoServer{}

	// setup tcp listener
	ds.tcp, err = net.Listen("tcp", TCPAddr)
	if err != nil {
		return ds, err
	}

	// setup udp
	laddr, _ := net.ResolveUDPAddr("udp", UDPAddr)
	ds.udp, err = net.ListenUDP("udp", laddr)
	if err != nil {
		return ds, err
	}

	// setup db
	const setupDB = `CREATE TABLE IF NOT EXISTS entries (id BLOB, addr TEXT, timestamp INTEGER)`
	const announce = ``
	const lookup = ``
	const getExpired = ``

	if _, err = ds.db.Exec(setupDB); err != nil {
		return ds, err
	}
	if ds.announce, err = ds.db.Prepare(announce); err != nil {
		return ds, err
	}
	if ds.getExpired, err = ds.db.Prepare(getExpired); err != nil {
		return ds, err
	}
	if ds.lookup, err = ds.db.Prepare(lookup); err != nil {
		return ds, err
	}

	// start gorountines
	go ds.cleanup()
	go ds.serveUDP()
	go ds.serveTCP()

	return ds, nil
}

func (ds *DiscoServer) serveTCP() {
	for {
		conn, err := ds.tcp.Accept()
		if err != nil {
			break
		}

		go func() {
			defer conn.Close()
			dec := json.NewDecoder(conn)
			enc := json.NewEncoder(conn)
			var m Message
			if err := dec.Decode(&m); err != nil {
				return
			}
			switch m.Type {
			case MessageAnnounce:
				m.Verify()

			case MessageLookup:
				reply := ds.Lookup(m.ID)
				enc.Encode(reply)
			}

		}()
	}
}

func (ds *DiscoServer) serveUDP() {
	buf := make([]byte, 1024)
	var m Message
	for {
		n, addr, err := ds.udp.ReadFrom(buf)
		if err != nil {
			break
		}

		go func() {
			if err := json.Unmarshal(buf[:n], &m); err != nil {
				return
			}
			switch m.Type {
			case MessageLookup:
				reply := ds.Lookup(m.ID)
				data, err := json.Marshal(reply)
				if err != nil {
					return
				}
				ds.udp.WriteTo(data, addr)

			case MessageAnnounce:
				if !m.Verify() {
					return
				}
				if _, err := ds.announce.Exec(m.ID, "udp", addr.String(), m.Timestamp.Unix()); err != nil {
					log.Println(err)
				}
			}
		}()
	}
}

//TODO: get all the expired rows and delete them
func (ds *DiscoServer) cleanup() {
	const cleanupInterval = time.Second * 2
	//del, err := ds.db.Prepare(``)
	//if err != nil {
	//	return
	//}

	for range time.Tick(cleanupInterval) {
		//rows, _ := ds.getExpired.Query()

	}
}

func (ds *DiscoServer) Lookup(id ed25519.PublicKey) *Reply {
	reply := &Reply{ID: id}
	rows, err := ds.lookup.Query(id)
	if err != nil {
		return reply
	}

	var addr *string
	for rows.Next() {
		rows.Scan(addr)
		reply.Addrs = append(reply.Addrs, *addr)
	}

	return reply
}

type Reply struct {
	ID    ed25519.PublicKey
	Addrs []string
}

func (m *Message) Verify() bool {
	//TODO: check timestamp and signature
	return true
}

func main() {
	ds, err := NewDiscoServer()
	fmt.Printf("%v\n %s\n", ds, err)
}
