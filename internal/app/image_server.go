package app

import (
	"strings"
	"sync"
)

type ImageServerService struct {
	mu   sync.RWMutex
	host string // host:port
	port string
}

func NewImageServerService(defaultHost, port string) *ImageServerService {
	h := strings.TrimSpace(defaultHost)
	p := strings.TrimSpace(port)
	if p == "" {
		p = "9003"
	}
	if h == "" {
		h = "localhost"
	}
	return &ImageServerService{
		host: h + ":" + p,
		port: p,
	}
}

func (s *ImageServerService) GetImgServerHost() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.host
}

func (s *ImageServerService) SetImgServerHost(server string) {
	server = strings.TrimSpace(server)
	if server == "" {
		return
	}
	s.mu.Lock()
	s.host = server + ":" + s.port
	s.mu.Unlock()
}
