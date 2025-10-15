package gateway

import (
	"sync"
)

type store struct {
	leaderNode   string
	leaderNodeIP string

	mu *sync.RWMutex
}

var s = store{
	mu: new(sync.RWMutex),
}

func (s *store) getLeaderNode() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.leaderNode
}

func (s *store) getLeaderNodeIP() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.leaderNodeIP
}

func (s *store) setLeaderNode(ip, hostname string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ip == "" || hostname == "" {
		return
	}
	s.leaderNode = hostname
	s.leaderNodeIP = ip
}

func LeaderNode() string {
	return s.getLeaderNode()
}

func LeaderNodeIP() string {
	return s.getLeaderNodeIP()
}

func SetLeaderNode(ip, hostname string) {
	s.setLeaderNode(ip, hostname)
}
