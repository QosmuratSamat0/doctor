package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
)

type NotifyRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	Channel        string `json:"channel"`
	Recipient      string `json:"recipient"`
	Message        string `json:"message"`
}

type NotifyResponse struct {
	Status string `json:"status"`
}

var (
	processedKeys = make(map[string]bool)
	mu            sync.Mutex
)

func main() {
	port := os.Getenv("GATEWAY_PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/notify", handleNotify)

	fmt.Printf("Mock Notification Gateway listening on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req NotifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Log request in JSON format to stdout
	logEntry, _ := json.Marshal(req)
	fmt.Println(string(logEntry))

	// Simulate 20% random failure (503)
	if rand.Float32() < 0.2 {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	status := "accepted"
	if processedKeys[req.IdempotencyKey] {
		status = "duplicate"
	} else {
		processedKeys[req.IdempotencyKey] = true
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NotifyResponse{Status: status})
}
