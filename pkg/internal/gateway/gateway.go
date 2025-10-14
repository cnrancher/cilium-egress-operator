package gateway

import (
	"maps"
	"sync"
)

type store struct {
	nodes map[string]string // map[NodeIP]Hostname

	mu *sync.RWMutex
}

var s = store{
	nodes: make(map[string]string),
	mu:    new(sync.RWMutex),
}

func (s *store) availableNodes() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return maps.Clone(s.nodes)
}

func (s *store) availableNode() (string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for ip, hostname := range s.nodes {
		if ip == "" || hostname == "" {
			continue
		}
		return ip, hostname
	}
	return "", ""
}

func (s *store) recordNode(ip, hostname string, available bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !available {
		delete(s.nodes, ip)
		return
	}
	if ip == "" || hostname == "" {
		return
	}
	s.nodes[ip] = hostname
}

func (s *store) nodeAvailable(ip, hostname string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if ip == "" || hostname == "" {
		return false
	}
	return s.nodes[ip] == hostname
}

func RecordNode(ip, hostname string, available bool) {
	s.recordNode(ip, hostname, available)
}

func NodeAvailable(ip, hostname string) bool {
	return s.nodeAvailable(ip, hostname)
}

func AvailableNodes() map[string]string {
	return s.availableNodes()
}

func AvailableNode() (string, string) {
	return s.availableNode()
}
