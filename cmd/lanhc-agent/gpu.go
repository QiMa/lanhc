// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

// GPU (compute) collector for lanhc-agent. Uses nvidia-smi when present so the
// hub can see accelerator count/utilization/temperature. Read-only and
// tolerant of missing tools: a host without NVIDIA drivers answers with an
// empty device list instead of failing the whole request.

package main

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// gpuQueryFields is the ordered column set we ask nvidia-smi for. Keeping the
// query explicit (rather than parsing the human table) makes the output stable
// across driver versions and locales.
var gpuQueryFields = []string{
	"index",
	"name",
	"uuid",
	"driver_version",
	"memory.total",
	"memory.used",
	"utilization.gpu",
	"utilization.memory",
	"temperature.gpu",
	"power.draw",
	"power.limit",
	"compute_cap",
}

type GPU struct {
	Index         int     `json:"index"`
	Name          string  `json:"name,omitempty"`
	UUID          string  `json:"uuid,omitempty"`
	DriverVersion string  `json:"driver_version,omitempty"`
	MemoryTotalMB int     `json:"memory_total_mb,omitzero"`
	MemoryUsedMB  int     `json:"memory_used_mb,omitzero"`
	UtilPct       float64 `json:"util_pct,omitzero"`
	MemUtilPct    float64 `json:"mem_util_pct,omitzero"`
	TemperatureC  float64 `json:"temperature_c,omitzero"`
	PowerW        float64 `json:"power_w,omitzero"`
	PowerLimitW   float64 `json:"power_limit_w,omitzero"`
	ComputeCap    string  `json:"compute_cap,omitempty"`
}

// GPUs is the payload returned by /v1/gpu. `GPUs` is always a (possibly empty)
// slice so callers can distinguish "no accelerators" from a failed read, which
// lands in Error instead.
type GPUs struct {
	GPUs        []GPU  `json:"gpus"`
	Count       int    `json:"count"`
	Driver      string `json:"driver,omitempty"`
	SMIVersion  string `json:"smi_version,omitempty"`
	Error       string `json:"error,omitempty"`
	CollectedAt string `json:"collected_at"`
}

// nvidiaSmiQuery prefers a PATH nvidia-smi (Linux/most hosts) and falls back to
// the WSL2 driver mount, where the binary is not on the default PATH of a
// non-interactive service.
func nvidiaSmiQuery() (string, bool) {
	args := []string{
		"--query-gpu=" + strings.Join(gpuQueryFields, ","),
		"--format=csv,noheader,nounits",
	}
	if lookPath("nvidia-smi") {
		out, err := runCommand("nvidia-smi", args...)
		return out, err == nil
	}
	for _, p := range []string{"/usr/lib/wsl/lib/nvidia-smi", "/usr/local/nvidia/bin/nvidia-smi"} {
		if _, err := os.Stat(p); err == nil {
			out, err := runCommand(p, args...)
			return out, err == nil
		}
	}
	return "", false
}

func collectGPUs() GPUs {
	res := GPUs{GPUs: []GPU{}, CollectedAt: time.Now().UTC().Format(time.RFC3339)}
	out, ok := nvidiaSmiQuery()
	if !ok {
		res.Error = "nvidia-smi not installed"
		return res
	}
	res.SMIVersion = parseSmiVersion(out)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		g, ok := parseGPUCSV(line)
		if !ok {
			continue
		}
		res.GPUs = append(res.GPUs, g)
	}
	if len(res.GPUs) == 0 {
		if res.Error == "" {
			res.Error = "nvidia-smi returned no devices"
		}
		return res
	}
	res.Count = len(res.GPUs)
	res.Driver = res.GPUs[0].DriverVersion
	res.Error = ""
	return res
}

// parseGPUCSV maps one `nvidia-smi --format=csv,noheader,nounits` row onto a
// GPU. Columns follow gpuQueryFields; unknown/NA cells are left zero so a
// driver that omits a field does not poison the whole scraped row.
func parseGPUCSV(line string) (GPU, bool) {
	parts := strings.Split(line, ",")
	if len(parts) < len(gpuQueryFields) {
		return GPU{}, false
	}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	if parts[0] == "" {
		return GPU{}, false
	}
	idx, err := strconv.Atoi(parts[0])
	if err != nil {
		return GPU{}, false
	}
	g := GPU{
		Index:         idx,
		Name:          naToEmpty(parts[1]),
		UUID:          naToEmpty(parts[2]),
		DriverVersion: naToEmpty(parts[3]),
		MemoryTotalMB: naToInt(parts[4]),
		MemoryUsedMB:  naToInt(parts[5]),
		UtilPct:       naToFloat(parts[6]),
		MemUtilPct:    naToFloat(parts[7]),
		TemperatureC:  naToFloat(parts[8]),
		PowerW:        naToFloat(parts[9]),
		PowerLimitW:   naToFloat(parts[10]),
		ComputeCap:    naToEmpty(parts[11]),
	}
	return g, true
}

func naToEmpty(s string) string {
	if s == "" || s == "[N/A]" || s == "N/A" {
		return ""
	}
	return s
}

func naToInt(s string) int {
	f := naToFloat(s)
	return int(f)
}

func naToFloat(s string) float64 {
	s = naToEmpty(s)
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

// parseSmiVersion reads the driver/CUDA banner that nvidia-smi prints before
// the queried rows, so the payload carries a version even with no devices.
func parseSmiVersion(out string) string {
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "Driver Version") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}
