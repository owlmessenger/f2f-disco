package inmembackend

import (
	"github.com/owlmessenger/f2f-disco/frontend"
	"sync"
	"time"
)

type Node struct {
	frontend.Entry

	next *Node
	prev *Node
}

func (n *Node) Remove() {
	n.next.prev = n.prev
	n.prev.next = n.next
}

type InMemBackend struct {
	mu     sync.RWMutex
	nodes  map[string][]*Node
	oldest *Node
	newest *Node
}

func New() *InMemBackend {
	imb := &InMemBackend{}
	imb.nodes = make(map[string][]*Node)
	return imb
}

func (imb *InMemBackend) remove(n *Node) {
	imb.mu.Lock()
	if imb.oldest == n {
		imb.oldest = n.next
	}
	if imb.newest == n {
		imb.newest = n.prev
	}
	if n.prev != nil {
		n.prev.next = n.next
	}
	if n.next != nil {
		n.next.prev = n.prev
	}
	imb.mu.Unlock()
}

func (imb *InMemBackend) append(n *Node) {
	imb.mu.Lock()
	if imb.newest == nil {
		imb.newest = n
		imb.oldest = n
	} else {
		imb.newest.next = n
		n.prev = imb.newest
	}
	imb.mu.Unlock()
}

func (imb *InMemBackend) Announce(entry frontend.Entry) {
	nslice, _ := imb.nodes[string(entry.ID)]
	for _, existing := range nslice {
		if existing.Addr.Host == entry.Addr.Host {
			existing.Timestamp = entry.Timestamp
			imb.remove(existing)
			imb.append(existing)
		}
	}
	n := &Node{}
	n.Entry = entry

	imb.nodes[string(entry.ID)] = append(nslice, n)
	imb.append(n)
}

func (imb *InMemBackend) Lookup(id []byte) (addrs []frontend.Addr) {
	nslice, _ := imb.nodes[string(id)]
	for _, n := range nslice {
		addrs = append(addrs, n.Addr)
	}
	return addrs
}

func (imb *InMemBackend) DeleteAfter(t time.Time) {
	for n := imb.oldest; n != nil && n.Timestamp.After(t); n = imb.oldest {
		imb.remove(n)
	}
}
