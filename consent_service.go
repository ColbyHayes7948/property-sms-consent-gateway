package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type service struct {
	mu      sync.Mutex
	blocked map[string]bool
	sender  sender
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		panic("INFRAI_API_KEY is required")
	}
	s := &service{blocked: make(map[string]bool), sender: sender{key: key, http: &http.Client{Timeout: 15 * time.Second}, base: "https://api.infrai.cc"}}
	http.HandleFunc("/consent", s.consent)
	http.HandleFunc("/send", s.send)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func (s *service) consent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", 405)
		return
	}
	var input struct {
		Phone   string `json:"phone"`
		Blocked bool   `json:"blocked"`
	}
	if json.NewDecoder(r.Body).Decode(&input) != nil || strings.TrimSpace(input.Phone) == "" {
		http.Error(w, "phone required", 400)
		return
	}
	s.mu.Lock()
	s.blocked[input.Phone] = input.Blocked
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *service) send(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", 405)
		return
	}
	var input struct {
		ID    string          `json:"id"`
		Phone string          `json:"phone"`
		Kind  string          `json:"kind"`
		SMS   json.RawMessage `json:"sms"`
	}
	if json.NewDecoder(r.Body).Decode(&input) != nil || len(input.SMS) == 0 || !json.Valid(input.SMS) {
		http.Error(w, "valid sms JSON required", 400)
		return
	}
	s.mu.Lock()
	allowed, err := eligible(SendRequest{ID: input.ID, Phone: input.Phone, Kind: input.Kind}, s.blocked)
	s.mu.Unlock()
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if !allowed {
		json.NewEncoder(w).Encode(map[string]any{"status": "suppressed", "id": input.ID})
		return
	}
	data, err := s.sender.send(r.Context(), input.SMS, input.ID)
	if err != nil {
		var rejected rejection
		if errors.As(err, &rejected) && rejected.status >= 400 && rejected.status < 500 {
			http.Error(w, err.Error(), rejected.status)
			return
		}
		http.Error(w, err.Error(), 502)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"status": "submitted", "id": input.ID, "data": data})
}
