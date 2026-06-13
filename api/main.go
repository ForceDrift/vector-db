package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	store := NewStore()

	addr := os.Getenv("VECTOR_DB_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	handler := apiHandler(store)

	port := parsePort(addr)
	fmt.Printf("Vector DB API listening on http://localhost:%s\n", port)
	fmt.Printf("Endpoints:\n")
	fmt.Printf("  POST /api/insert   — insert a vector\n")
	fmt.Printf("  POST /api/remove   — remove a vector\n")
	fmt.Printf("  POST /api/get      — get a vector\n")
	fmt.Printf("  POST /api/search   — search vectors\n")
	fmt.Printf("  GET  /api/health   — health check\n")

	if err := http.ListenAndServe(addr, handler); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
