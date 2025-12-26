package server

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

type SimpleServer struct {
	addr  string
	proxy *httputil.ReverseProxy
	alive bool
	mu    sync.RWMutex //why RWMutex?
}

func NewServer(addr string) *SimpleServer {
	serverUrl, err := url.Parse(addr)
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(serverUrl)

	s := &SimpleServer{
		addr:  addr,
		proxy: proxy,
		alive: true,
	}
	//if server doesn't respond in 2 seconds, consider it dead
	//so we are adding this beacause it might be possible that server is slow and we don't want to wait indefinitely
	proxy.Transport = &http.Transport{
		ResponseHeaderTimeout: 2 * time.Second,
	}

	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		s.mu.Lock()
		s.alive = false
		s.mu.Unlock()

		fmt.Printf("error while proxying to %s: %v\n", addr, err)
	}

	s.proxy = proxy

	return s
}

func (s *SimpleServer) Address() string {
	return s.addr
}

func (s *SimpleServer) IsAlive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.alive
}

func (s *SimpleServer) CheckHealth() {
	timeout := 2 * time.Second

	client := http.Client{
		Timeout: timeout,
	}

	resp, err := client.Get(s.addr + "/health")
	if err != nil {
		s.alive = false
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.alive = false
		return
	}

	s.alive = true
}

func (s *SimpleServer) Serve(rw http.ResponseWriter, req *http.Request) {
	s.proxy.ServeHTTP(rw, req)
}
