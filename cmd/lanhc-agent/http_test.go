// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func get(t *testing.T, path string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	var body map[string]any
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("invalid JSON from %s: %v (%s)", path, err, rec.Body.String())
		}
	}
	return rec, body
}

func TestInventoryEndpoint(t *testing.T) {
	rec, body := get(t, "/v1/inventory")
	if rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}
	if body["hostname"] == nil || body["cpu_count"] == nil {
		t.Fatalf("inventory missing fields: %v", body)
	}
}

func TestHealthEndpoint(t *testing.T) {
	rec, body := get(t, "/v1/health")
	if rec.Code != 200 || body["collected_at"] == nil {
		t.Fatalf("unexpected health response: %d %v", rec.Code, body)
	}
}

func TestSmartRequiresDevice(t *testing.T) {
	rec, body := get(t, "/v1/disk/smart")
	if rec.Code != 400 || !strings.Contains(body["error"].(string), "dev") {
		t.Fatalf("expected 400 with dev hint, got %d %v", rec.Code, body)
	}
}

func TestLogsDefaultScope(t *testing.T) {
	rec, body := get(t, "/v1/logs")
	if rec.Code != 200 || body["scope"] != "dmesg" {
		t.Fatalf("expected dmesg default, got %d %v", rec.Code, body)
	}
}

func TestLogsUnknownScope(t *testing.T) {
	_, body := get(t, "/v1/logs?scope=rm-rf")
	if body["error"] == nil {
		t.Fatalf("expected error for unknown scope, got %v", body)
	}
}

func TestExecRejectsFreeform(t *testing.T) {
	payload := `{"template_id":"smartctl-info","param":"device","value":"/dev/sda; reboot"}`
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/exec", strings.NewReader(payload)))
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "forbidden") {
		t.Fatalf("expected 400 forbidden, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestExecRejectsUnknownTemplate(t *testing.T) {
	payload := `{"template_id":"sh","param":"device","value":"/dev/sda"}`
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/exec", strings.NewReader(payload)))
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "unknown command template") {
		t.Fatalf("expected 400 unknown template, got %d %s", rec.Code, rec.Body.String())
	}
}
