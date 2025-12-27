package balancer

import (
	"bytes"
	"context"
	"fmt"
	"io"
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

func cloneRequest(r *http.Request, body []byte) *http.Request {
	req := r.Clone(r.Context())
	if body != nil {
		req.Body = io.NopCloser(bytes.NewBuffer(body))
	}
	return req
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

	var body []byte
	var err error

	if r.Body != nil {
		body, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
	}

	for i := 0; i < maxRetries; i++ {
		srv := lb.getNextAvailableServer()
		if srv != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer cancel()

			req := cloneRequest(r.WithContext(ctx), body)
			fmt.Println("Forwarding request to:", srv.Address())
			srv.Serve(w, req)
			return
		}
	}

	http.Error(w, "All backends unavailable", http.StatusServiceUnavailable)
}

func (lb *LoadBalancer) Port() string {
	return lb.port
}
