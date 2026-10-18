package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEligible(t *testing.T) {
	cases := []struct {
		name, kind, phone string
		blocked           bool
		want              bool
		invalid           bool
	}{
		{"maintenance allowed", "maintenance_request", "+15550000001", false, true, false},
		{"document opted out", "tenant_document", "+15550000001", true, false, false},
		{"inspection suppressed", "inspection_reminder", "+15550000002", true, false, false},
		{"unknown event", "advertisement", "+15550000001", false, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := eligible(SendRequest{ID: "event-42", Phone: tc.phone, Kind: tc.kind}, map[string]bool{tc.phone: tc.blocked})
			if got != tc.want || (err != nil) != tc.invalid {
				t.Fatalf("eligible = %v, err = %v", got, err)
			}
		})
	}
}

func TestSenderIncludesIdempotencyKeyInRequestBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Idempotency-Key"); got != "" {
			t.Errorf("unexpected Idempotency-Key header %q", got)
		}
		var body struct {
			Messages       json.RawMessage `json:"messages"`
			IdempotencyKey string          `json:"idempotency_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if string(body.Messages) != `[]` || body.IdempotencyKey != "event-42" {
			t.Errorf("request body = messages %s, idempotency_key %q", body.Messages, body.IdempotencyKey)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"data":{}}`))
	}))
	defer server.Close()

	s := sender{key: "test", http: server.Client(), base: server.URL}
	if _, err := s.send(context.Background(), json.RawMessage(`{"messages":[]}`), "event-42"); err != nil {
		t.Fatal(err)
	}
}
