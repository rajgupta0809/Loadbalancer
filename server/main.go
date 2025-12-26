package main

import (
	"fmt"
	"net/http"
)

// a sample server to demonstrate health check endpoint
func main() {
	http.HandleFunc("/health", healthCheckHandler)
	http.ListenAndServe(":8000", nil)
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}
