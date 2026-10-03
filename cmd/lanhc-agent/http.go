// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"lanhc.com/util/httpm"
)

// newMux wires the read-only agent API. Every endpoint is intentionally GET
// (except the template exec POST) so it is easy to reason about and to gate.
func newMux() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":           true,
			"role":         "lanhc-agent",
			"version":      version,
			"collected_at": time.Now().UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("/v1/inventory", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, collectInventory())
	})

	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, collectHealth())
	})

	mux.HandleFunc("/v1/disk/list", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, collectDiskList())
	})

	mux.HandleFunc("/v1/disk/smart", func(w http.ResponseWriter, r *http.Request) {
		device := r.URL.Query().Get("dev")
		if device == "" {
			writeErr(w, http.StatusBadRequest, "dev query parameter is required, e.g. /dev/sda")
			return
		}
		deviceType := r.URL.Query().Get("type")
		writeJSON(w, http.StatusOK, collectSMART(device, deviceType))
	})

	mux.HandleFunc("/v1/logs", func(w http.ResponseWriter, r *http.Request) {
		scope := r.URL.Query().Get("scope")
		if scope == "" {
			scope = "dmesg"
		}
		tail, _ := strconv.Atoi(r.URL.Query().Get("tail"))
		if tail <= 0 || tail > 5000 {
			tail = 500
		}
		writeJSON(w, http.StatusOK, collectLogs(scope, tail))
	})

	mux.HandleFunc("/v1/exec", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != httpm.POST {
			writeErr(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req struct {
			TemplateID string `json:"template_id"`
			Param      string `json:"param"`
			Value      string `json:"value"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
			return
		}
		out, err := runTemplate(req.TemplateID, req.Param, req.Value)
		if err != nil && out == "" {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if err != nil {
			log.Printf("exec template %s: %v", req.TemplateID, err)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"template_id": req.TemplateID,
			"ok":          err == nil,
			"output":      out,
		})
	})

	return mux
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]any{"error": msg})
}
