package main

import (
	"fmt"
	"net/http"
	"time"

	"loadbalancer/internal/balancer"
	"loadbalancer/internal/server"
)

func main() {

	servers := []server.Server{
		server.NewServer("https://google.com"),
		server.NewServer("https://facebook.com"),
		server.NewServer("http://localhost:8000"),
	}

	lb := balancer.NewLoadBalancer("8080", servers)

	go lb.HealthCheck(10 * time.Second)

	http.HandleFunc("/", lb.ServeProxy)

	fmt.Println("server started at :8080")
	err := http.ListenAndServe(":"+lb.Port(), nil)
	if err != nil {
		fmt.Println("error starting server:", err)
	}
}
