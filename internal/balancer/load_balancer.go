package balancer

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"loadbalancer/internal/server"
)

type LoadBalancer struct {
	port    string
	servers []server.Server
	counter atomic.Int64
}

func NewLoadBalancer(port string, servers []server.Server) *LoadBalancer {
	return &LoadBalancer{
		port:    port,
		servers: servers,
	}
}

func (lb *LoadBalancer) HealthCheck(interval time.Duration) {
	ticker := time.NewTicker(interval)

	for range ticker.C {
		for _, srv := range lb.servers {
			if s, ok := srv.(*server.SimpleServer); ok {
				go s.CheckHealth()
			}
		}
	}
}

func (lb *LoadBalancer) getNextAvailableServer() server.Server {
	total := len(lb.servers)
	if total == 0 {
		return nil
	}

	start := lb.counter.Add(1) - 1

	for i := 0; i < total; i++ {
		idx := int((start + int64(i)) % int64(total))
		srv := lb.servers[idx]
		if srv.IsAlive() {
			return srv
		}
	}
	return nil
}

func (lb *LoadBalancer) ServeProxy(w http.ResponseWriter, r *http.Request) {
	const maxRetries = 3

	for i := 0; i < maxRetries; i++ {
		srv := lb.getNextAvailableServer()
		if srv != nil {
			fmt.Println("Forwarding request to:", srv.Address())
			srv.Serve(w, r)
			return
		}
	}

	http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
}

func (lb *LoadBalancer) Port() string {
	return lb.port
}
