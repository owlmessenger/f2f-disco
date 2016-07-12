package frontend

import (
	"encoding/json"
	"net"
	"time"
)

const (
	UDPAddr = "8001"
	TCPAddr = ":8000"
)

type DiscoServer struct {
	tcp     net.Listener
	udp     *net.UDPConn
	backend Backend
}

func NewDiscoServer(backend Backend) (ds *DiscoServer, err error) {
	ds = &DiscoServer{backend: backend}

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

	return ds, nil
}

func (ds *DiscoServer) Serve() error {
	errch := make(chan error, 3)
	go ds.cleanup(errch)
	go ds.serveUDP(errch)
	go ds.serveTCP(errch)
	return <-errch
}

func (ds *DiscoServer) serveTCP(errch chan error) {
	for {
		conn, err := ds.tcp.Accept()
		if err != nil {
			errch <- err
			return
		}

		go func() {
			defer conn.Close()
			dec := json.NewDecoder(conn)
			enc := json.NewEncoder(conn)
			var m Message
			var reply Reply
			var entry Entry

			if err := dec.Decode(&m); err != nil {
				return
			}

			switch m.Type {
			case MessageAnnounce:
				m.Verify()
				entry.Addr.Protocol = "tcp"
				entry.Addr.Host = conn.RemoteAddr().String()
				entry.ID = m.ID
				ds.backend.Announce(entry)

			case MessageLookup:
				reply.Addrs = ds.backend.Lookup(m.ID)
				reply.ID = m.ID
				enc.Encode(reply)
			}

		}()
	}
}

func (ds *DiscoServer) serveUDP(errch chan error) {
	buf := make([]byte, 1024)
	for {
		n, addr, err := ds.udp.ReadFrom(buf)
		if err != nil {
			errch <- err
			return
		}
		var m Message
		if err := json.Unmarshal(buf[:n], &m); err != nil {
			continue
		}

		go func() {
			var reply Reply
			var entry Entry

			switch m.Type {
			case MessageLookup:
				reply.Addrs = ds.backend.Lookup(m.ID)
				reply.ID = m.ID
				data, err := json.Marshal(reply)
				if err != nil {
					return
				}
				ds.udp.WriteTo(data, addr)

			case MessageAnnounce:
				if !m.Verify() {
					return
				}
				entry.Addr.Protocol = "udp"
				entry.Addr.Host = addr.String()

				ds.backend.Announce(entry)
			}
		}()
	}
}

func (ds *DiscoServer) cleanup(errch chan error) {
	const cleanupInterval = time.Second * 2

	for t := range time.Tick(cleanupInterval) {
		ds.backend.DeleteAfter(t)
	}
}
