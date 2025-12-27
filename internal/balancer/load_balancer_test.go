package balancer

import (
	"loadbalancer/internal/server"
	"net/http"
	"testing"
)

type fakeServer struct {
	addr  string
	alive bool
}

func (f *fakeServer) Address() string {
	return f.addr
}

func (f *fakeServer) IsAlive() bool {
	return f.alive
}

func (f *fakeServer) Serve(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestGetNextAvailableServer_SkipsUnhealthy(t *testing.T) {
	s1 := &fakeServer{addr: "s1", alive: false}
	s2 := &fakeServer{addr: "s2", alive: true}

	lb := NewLoadBalancer("8080", []server.Server{s1, s2})

	server := lb.getNextAvailableServer()

	if server == nil {
		t.Fatal("expected a server, got nil")
	}

	if server.Address() != "s2" {
		t.Fatalf("expected s2, got %s", server.Address())
	}
}
