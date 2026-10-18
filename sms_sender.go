package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type reply struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type sender struct {
	key  string
	http *http.Client
	base string
}

type rejection struct {
	status  int
	message string
}

func (e rejection) Error() string { return e.message }

func (s sender) send(ctx context.Context, payload json.RawMessage, id string) (json.RawMessage, error) {
	var params map[string]json.RawMessage
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("decode sms request: %w", err)
	}
	idempotencyKey, err := json.Marshal(id)
	if err != nil {
		return nil, fmt.Errorf("encode idempotency key: %w", err)
	}
	params["idempotency_key"] = idempotencyKey
	payload, err = json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("encode sms request: %w", err)
	}

	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.base+"/v1/sms/batch/send", bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+s.key)
		req.Header.Set("Content-Type", "application/json")
		res, err := s.http.Do(req)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		res.Body.Close()
		if err != nil {
			return nil, err
		}
		var env reply
		if err := json.Unmarshal(body, &env); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := time.Duration(1<<attempt) * time.Second
			if seconds, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && seconds >= 0 {
				delay = time.Duration(seconds) * time.Second
			}
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
			continue
		}
		if !env.OK {
			return nil, rejection{status: res.StatusCode, message: fmt.Sprintf("sms rejected: %s", env.Error)}
		}
		if res.StatusCode >= 400 {
			return nil, fmt.Errorf("sms HTTP %d", res.StatusCode)
		}
		return env.Data, nil
	}
	return nil, fmt.Errorf("sms retry limit reached")
}
