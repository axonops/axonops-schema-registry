// Package main implements a simple HTTP server for testing webhook audit output.
// It stores received events in memory and exposes them via a test API.
//
// Endpoints:
//
//	POST /           — receive audit events (newline-delimited JSON)
//	GET  /events     — return all stored events as a JSON array
//	DELETE /events   — clear all stored events
//	POST /shutdown   — start returning 503 (simulate outage)
//	POST /reset      — stop returning 503
//	GET  /health     — health check (200 or 503)
package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

type server struct {
	mu       sync.Mutex
	events   []string
	shutdown bool
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}

	s := &server{}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /", s.handleReceive)
	mux.HandleFunc("GET /events", s.handleGetEvents)
	mux.HandleFunc("DELETE /events", s.handleClearEvents)
	mux.HandleFunc("POST /shutdown", s.handleShutdown)
	mux.HandleFunc("POST /reset", s.handleReset)
	mux.HandleFunc("GET /health", s.handleHealth)

	log.Printf("webhook-receiver listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func (s *server) handleReceive(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	if s.shutdown {
		s.mu.Unlock()
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	s.mu.Unlock()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
		if line != "" {
			s.events = append(s.events, line)
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (s *server) handleGetEvents(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.events) //nolint:errcheck
}

func (s *server) handleClearEvents(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = nil
	w.WriteHeader(http.StatusOK)
}

func (s *server) handleShutdown(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shutdown = true
	w.WriteHeader(http.StatusOK)
}

func (s *server) handleReset(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shutdown = false
	w.WriteHeader(http.StatusOK)
}

func (s *server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.shutdown {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}
