package cidr

import (
	"net"
	"sync"
)

// Store provides storage and lookup for a list of CIDRs.
type Store struct {
	mu   sync.RWMutex
	nets []*net.IPNet
}

// NewStore creates an empty CIDR store.
func NewStore() *Store {
	return &Store{
		nets: make([]*net.IPNet, 0),
	}
}

// Set atomically replaces the current list of subnets.
func (s *Store) Set(nets []*net.IPNet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nets = nets
}

// Contains checks whether an IP belongs to any of the subnets.
func (s *Store) Contains(ip net.IP) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, n := range s.nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
