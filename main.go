package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type Server interface {
	Address() string
	IsAlive() bool
	Serve(rw http.ResponseWriter, req *http.Request)
}

type simpleServer struct {
	addr  string
	proxy *httputil.ReverseProxy
	alive bool
	mu    sync.RWMutex //why RWMutex?
}

func NewServer(addr string) *simpleServer {
	serverUrl, err := url.Parse(addr)
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(serverUrl)

	s := &simpleServer{
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

func (s *simpleServer) Address() string {
	return s.addr
}

func (s *simpleServer) IsAlive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.alive
}

func (s *simpleServer) CheckHealth() {
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

func (s *simpleServer) Serve(rw http.ResponseWriter, req *http.Request) {
	s.proxy.ServeHTTP(rw, req)
}

type LoadBalancer struct {
	port    string
	servers []Server
	counter atomic.Int64
}

func NewLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		port:    port,
		servers: servers,
	}
}

func (lb *LoadBalancer) healthCheck(interval time.Duration) { //explain
	ticker := time.NewTicker(interval)

	for range ticker.C {
		for _, server := range lb.servers {
			go server.(*simpleServer).CheckHealth()
		}
	}
}

func (lb *LoadBalancer) getNextAvailableServer() Server {
	totalServers := len(lb.servers)
	if totalServers == 0 {
		return nil
	}

	start := lb.counter.Add(1) - 1

	for i := 0; i < totalServers; i++ {
		index := int64(start+int64(i)) % int64(totalServers)
		server := lb.servers[index]
		if server.IsAlive() {
			return server
		}
	}
	return nil
}

func (lb *LoadBalancer) ServeProxy(rw http.ResponseWriter, req *http.Request) {
	const maxRetries = 3
	for i := 0; i < maxRetries; i++ {
		targetServer := lb.getNextAvailableServer()
		if targetServer != nil {
			fmt.Println("Forwarding request to:", targetServer.Address())
			targetServer.Serve(rw, req)
			return
		}
	}

	http.Error(rw, "Service not available", http.StatusServiceUnavailable)
}

func main() {

	servers := []Server{
		NewServer("https://google.com"),
		NewServer("https://facebook.com"),
		NewServer("http://localhost:8000"),
	}

	loadBalancer := NewLoadBalancer("8080", servers)
	go loadBalancer.healthCheck(10 * time.Second)

	handleRedirect := func(w http.ResponseWriter, req *http.Request) {
		loadBalancer.ServeProxy(w, req)
	}

	fmt.Println("server started at :8080")

	http.HandleFunc("/", handleRedirect)
	err := http.ListenAndServe(":"+loadBalancer.port, nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
