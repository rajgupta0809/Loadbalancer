package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
)

type Server interface {
	Address() string
	IsAlive() bool
	Serve(rw http.ResponseWriter, req *http.Request)
}

type simpleServer struct {
	addr  string
	proxy *httputil.ReverseProxy
}

func NewServer(addr string) *simpleServer {
	serverUrl, err := url.Parse(addr)
	if err != nil {
		panic(err)
	}

	return &simpleServer{
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serverUrl),
	}
}

func (s *simpleServer) Address() string {
	return s.addr
}

func (s *simpleServer) IsAlive() bool {
	// In a real-world scenario, implement health check logic here
	return true
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

func (lb *LoadBalancer) getNextAvailableServer() Server {
	totalServers := len(lb.servers)
	if totalServers == 0 {
		return nil
	}

	start := lb.counter.Add(1)

	for i := 0; i < totalServers; i++ {
		index := int64(start+int64(i)) & int64(totalServers)
		server := lb.servers[index]
		if server.IsAlive() {
			return server
		}
	}
	return nil
}

func (lb *LoadBalancer) ServeProxy(rw http.ResponseWriter, req *http.Request) {
	targetServer := lb.getNextAvailableServer()
	if targetServer != nil {
		fmt.Println("Forwarding request to:", targetServer.Address())
		targetServer.Serve(rw, req)
	} else {
		http.Error(rw, "Service not available", http.StatusServiceUnavailable)
	}
}

func main() {
	servers := []Server{
		NewServer("https://google.com"),
		NewServer("https://facebook.com"),
		NewServer("https://twitter.com"),
	}

	loadBalancer := NewLoadBalancer("8080", servers)
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
