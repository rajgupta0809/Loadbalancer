package main

import (
	"fmt"
	"net/http"
	"time"
)

// a sample server to demonstrate health check endpoint
func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // simulate some processing delay
		fmt.Fprintln(w, "Hello from the backend server!")
	})

	http.HandleFunc("/health", healthCheckHandler)
	fmt.Println("server is listening on :8000")
	http.ListenAndServe(":8000", nil)
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}
