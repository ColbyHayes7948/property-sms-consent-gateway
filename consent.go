package main

import (
	"errors"
	"strings"
)

type SendRequest struct {
	ID      string `json:"id"`
	Phone   string `json:"phone"`
	Kind    string `json:"kind"`
	Payload []byte `json:"-"`
}

var kinds = map[string]bool{"maintenance_request": true, "tenant_document": true, "inspection_reminder": true}

func eligible(r SendRequest, blocked map[string]bool) (bool, error) {
	if strings.TrimSpace(r.ID) == "" || strings.TrimSpace(r.Phone) == "" || !kinds[r.Kind] {
		return false, errors.New("id, phone and a supported kind are required")
	}
	return !blocked[r.Phone], nil
}
