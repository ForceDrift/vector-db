package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if err := os.MkdirAll("data", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create data dir: %v\n", err)
		os.Exit(1)
	}

	store := NewStore()
	defer store.Close()

	addr := os.Getenv("VECTOR_DB_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	handler := apiHandler(store)

	port := parsePort(addr)
	fmt.Printf("Vector DB API (C++ engine) listening on http://localhost:%s\n", port)
	fmt.Printf("Endpoints:\n")
	fmt.Printf("  POST /api/insert          — insert a vector\n")
	fmt.Printf("  POST /api/remove          — remove a vector\n")
	fmt.Printf("  POST /api/get             — get a vector\n")
	fmt.Printf("  GET  /api/getall          — list all vectors\n")
	fmt.Printf("  POST /api/search          — search vectors\n")
	fmt.Printf("  POST /api/persist/save    — checkpoint (snapshot + truncate WAL)\n")
	fmt.Printf("  POST /api/persist/load    — reload from disk\n")
	fmt.Printf("  GET  /api/health          — health check\n")
	fmt.Printf("\nData directory: data/\n")

	// Graceful shutdown: checkpoint on SIGINT/SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down, saving checkpoint...")
		store.SaveSnapshot()
		store.Close()
		os.Exit(0)
	}()

	if err := http.ListenAndServe(addr, handler); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
