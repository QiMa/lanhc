// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"net"
	"testing"
)

// Mirrors real `nvidia-smi --query-gpu=... --format=csv,noheader,nounits`
// output on the canary host (WSL2, RTX 3080).
const gpuCSV = `0, NVIDIA GeForce RTX 3080, GPU-3cad2281-d63f-792e-0b2d-b2522fe7db31, 591.86, 10240, 1477, 30, 12, 43, 27.25, 320.00, 8.6`

func TestParseGPUCSV(t *testing.T) {
	g, ok := parseGPUCSV(gpuCSV)
	if !ok {
		t.Fatal("expected row to parse")
	}
	if g.Index != 0 || g.Name != "NVIDIA GeForce RTX 3080" {
		t.Fatalf("identity wrong: %+v", g)
	}
	if g.MemoryTotalMB != 10240 || g.MemoryUsedMB != 1477 {
		t.Fatalf("memory wrong: %+v", g)
	}
	if g.UtilPct != 30 || g.TemperatureC != 43 || g.PowerW != 27.25 {
		t.Fatalf("telemetry wrong: %+v", g)
	}
	if g.ComputeCap != "8.6" || g.DriverVersion != "591.86" {
		t.Fatalf("versions wrong: %+v", g)
	}
}

// A driver that reports [N/A] for a field must not zero out the other columns,
// and a truncated row must be rejected rather than half-parsed.
func TestParseGPUCSVNAAndJunk(t *testing.T) {
	g, ok := parseGPUCSV("1, NVIDIA H100, GPU-x, 555.1, 81920, [N/A], 0, [N/A], 51, [N/A], 700, 9.0")
	if !ok {
		t.Fatal("expected NA row to parse")
	}
	if g.MemoryUsedMB != 0 || g.PowerW != 0 {
		t.Fatalf("NA should map to zero: %+v", g)
	}
	if g.ComputeCap != "9.0" {
		t.Fatalf("later columns must survive NA: %+v", g)
	}
	if _, ok := parseGPUCSV("0, only, three, columns"); ok {
		t.Fatal("short row must be rejected")
	}
	if _, ok := parseGPUCSV("not-an-index, x, y, z, 1, 1, 1, 1, 1, 1, 1, 1"); ok {
		t.Fatal("non-numeric index must be rejected")
	}
}

func TestIsTailnetIP(t *testing.T) {
	for ip, want := range map[string]bool{
		"100.64.0.2":        true,
		"100.127.255.254":   true,
		"192.168.1.2":       false,
		"172.18.0.74":       false,
		"fd7a:115c:a1e0::2": true,
		"fe80::1":           false,
		"10.0.0.1":          false,
	} {
		if got := isTailnetIP(net.ParseIP(ip)); got != want {
			t.Fatalf("isTailnetIP(%s)=%v want %v", ip, got, want)
		}
	}
}
