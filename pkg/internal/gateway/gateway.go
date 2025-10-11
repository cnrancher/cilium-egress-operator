package gateway

import (
	"slices"
	"sync"
)

type store struct {
	nodeIPs map[string]bool

	mu *sync.RWMutex
}

var s = store{
	nodeIPs: make(map[string]bool),
	mu:      new(sync.RWMutex),
}

func (s *store) availableIPs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ips := []string{}
	for ip, ok := range s.nodeIPs {
		if !ok {
			continue
		}
		ips = append(ips, ip)
	}
	slices.Sort(ips)
	return ips
}

func (s *store) recordNodeIP(ip string, available bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodeIPs[ip] = available
}

func (s *store) nodeAvailable(ip string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.nodeIPs[ip]
}

func RecordNodeIP(ip string, available bool) {
	s.recordNodeIP(ip, available)
}

func NodeAvailable(ip string) bool {
	return s.nodeAvailable(ip)
}

func GetAvailableIPs() []string {
	return s.availableIPs()
}
