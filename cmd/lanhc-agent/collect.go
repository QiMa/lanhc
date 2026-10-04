// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

// Collectors for lanhc-agent. Every collector is read-only and must tolerate
// missing tools: a server without smartctl or journalctl should still answer
// inventory/health instead of failing the whole request.

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// execTimeout bounds every external command so a wedged device cannot hang the agent.
const execTimeout = 20 * time.Second

type Inventory struct {
	Hostname     string            `json:"hostname"`
	OS           string            `json:"os"`
	Kernel       string            `json:"kernel,omitempty"`
	Arch         string            `json:"arch"`
	UptimeSec    float64           `json:"uptime_sec,omitzero"`
	CPUCount     int               `json:"cpu_count"`
	MemTotalKB   int64             `json:"mem_total_kb,omitzero"`
	LanhcIPs     []string          `json:"lanhc_ips,omitempty"`
	LocalIPs     []string          `json:"local_ips,omitempty"`
	LanhcBackend string            `json:"lanhc_backend,omitempty"`
	Disks        []Disk            `json:"disks,omitempty"`
	CollectedAt  string            `json:"collected_at"`
	Extra        map[string]string `json:"extra,omitempty"`
}

type Disk struct {
	Name       string  `json:"name"`
	SizeGB     float64 `json:"size_gb,omitzero"`
	Rotational *bool   `json:"rotational,omitzero"`
}

type Health struct {
	LoadAvg     []float64 `json:"load_avg,omitempty"`
	MemUsedPct  float64   `json:"mem_used_pct,omitzero"`
	SwapUsedPct float64   `json:"swap_used_pct,omitzero"`
	RootUsedPct float64   `json:"root_used_pct,omitzero"`
	FailedUnits []string  `json:"failed_units,omitempty"`
	Errors      []string  `json:"errors,omitempty"`
	CollectedAt string    `json:"collected_at"`
}

func runCommand(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), execTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func lookPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func readFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func collectInventory() Inventory {
	hostname, _ := os.Hostname()
	inv := Inventory{
		Hostname:    hostname,
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		CPUCount:    runtime.NumCPU(),
		CollectedAt: time.Now().UTC().Format(time.RFC3339),
		Extra:       map[string]string{},
	}
	if u := readFile("/proc/uptime"); u != "" {
		if f := strings.Fields(u); len(f) > 0 {
			if v, err := strconv.ParseFloat(f[0], 64); err == nil {
				inv.UptimeSec = v
			}
		}
	}
	if rel := readFile("/proc/sys/kernel/osrelease"); rel != "" {
		inv.Kernel = strings.TrimSpace(rel)
	}
	if mem := readFile("/proc/meminfo"); mem != "" {
		for _, line := range strings.Split(mem, "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				inv.MemTotalKB = parseKB(line)
				break
			}
		}
	}
	inv.Disks = collectDisks()
	inv.LocalIPs = collectLocalIPs()
	return inv
}

// tailnetNet is the lanhc/headscale CGNAT range (100.64.0.0/10) plus the
// fd7a:115c:a1e0::/48 ULA prefix. Those addresses are already shown as the
// node's tailnet IP; localIPs is specifically the *other* addresses a device
// holds on its LAN, which is what an operator needs to reach it out of band.
var (
	tailnetV4 = mustCIDR("100.64.0.0/10")
	tailnetV6 = mustCIDR("fd7a:115c:a1e0::/48")
)

func mustCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		return nil
	}
	return n
}

func isTailnetIP(ip net.IP) bool {
	if tailnetV4 != nil && tailnetV4.Contains(ip) {
		return true
	}
	return tailnetV6 != nil && tailnetV6.Contains(ip)
}

// collectLocalIPs returns non-loopback, non-tailnet addresses from the host's
// interfaces. Interfaces the agent cannot inspect are skipped, never fatal.
func collectLocalIPs() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil || ip == nil {
				continue
			}
			if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
				continue
			}
			if isTailnetIP(ip) {
				continue
			}
			out = append(out, ip.String())
		}
	}
	return out
}

func parseKB(line string) int64 {
	f := strings.Fields(line)
	if len(f) < 2 {
		return 0
	}
	v, _ := strconv.ParseInt(f[1], 10, 64)
	return v
}

// collectDisks reads /sys/block without shelling out, so it works on minimal hosts.
func collectDisks() []Disk {
	entries, err := os.ReadDir("/sys/block")
	if err != nil {
		return nil
	}
	var disks []Disk
	for _, e := range entries {
		name := e.Name()
		// Skip virtual/loop/ram devices; we care about real storage.
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") ||
			strings.HasPrefix(name, "dm-") || strings.HasPrefix(name, "md") {
			continue
		}
		d := Disk{Name: "/dev/" + name}
		if sectors := strings.TrimSpace(readFile("/sys/block/" + name + "/size")); sectors != "" {
			if v, err := strconv.ParseInt(sectors, 10, 64); err == nil {
				d.SizeGB = float64(v) * 512 / (1000 * 1000 * 1000)
			}
		}
		if rot := strings.TrimSpace(readFile("/sys/block/" + name + "/queue/rotational")); rot != "" {
			b := rot == "1"
			d.Rotational = &b
		}
		disks = append(disks, d)
	}
	return disks
}

func collectHealth() Health {
	h := Health{CollectedAt: time.Now().UTC().Format(time.RFC3339)}
	if la := readFile("/proc/loadavg"); la != "" {
		f := strings.Fields(la)
		for i := 0; i < 3 && i < len(f); i++ {
			if v, err := strconv.ParseFloat(f[i], 64); err == nil {
				h.LoadAvg = append(h.LoadAvg, v)
			}
		}
	}
	h.MemUsedPct, h.SwapUsedPct = memUsedPct()
	h.RootUsedPct = diskUsedPct("/")
	if lookPath("systemctl") {
		if out, err := runCommand("systemctl", "--failed", "--no-legend", "--plain"); err == nil {
			for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
				if line = strings.TrimSpace(line); line != "" {
					h.FailedUnits = append(h.FailedUnits, line)
				}
			}
		} else {
			h.Errors = append(h.Errors, "systemctl --failed: "+err.Error())
		}
	}
	return h
}

func memUsedPct() (mem, swap float64) {
	total, avail, swapTotal, swapFree := int64(0), int64(0), int64(0), int64(0)
	sc := bufio.NewScanner(strings.NewReader(readFile("/proc/meminfo")))
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "MemTotal:"):
			total = parseKB(line)
		case strings.HasPrefix(line, "MemAvailable:"):
			avail = parseKB(line)
		case strings.HasPrefix(line, "SwapTotal:"):
			swapTotal = parseKB(line)
		case strings.HasPrefix(line, "SwapFree:"):
			swapFree = parseKB(line)
		}
	}
	if total > 0 {
		mem = 100 * float64(total-avail) / float64(total)
	}
	if swapTotal > 0 {
		swap = 100 * float64(swapTotal-swapFree) / float64(swapTotal)
	}
	return mem, swap
}

// collectDiskList returns the physical devices smartctl can actually address,
// including RAID members behind a PERC/DELL controller. The block devices from
// /sys/block are not enough on such hosts: /dev/sda is a virtual volume, while
// the real SMART targets live behind /dev/bus/0 with "-d megaraid,N".
func collectDiskList() map[string]any {
	res := map[string]any{"collected_at": time.Now().UTC().Format(time.RFC3339)}
	disks := []map[string]string{}
	if !lookPath("smartctl") {
		res["disks"] = disks
		res["error"] = "smartctl not installed"
		return res
	}
	out, err := runCommand("smartctl", "--scan-open")
	res["scan_open"] = out
	if err != nil {
		res["disks"] = disks
		res["error"] = "smartctl --scan-open failed: " + err.Error()
		return res
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		dev := ""
		typ := ""
		label := ""
		if i := strings.Index(line, " -d "); i >= 0 {
			dev = strings.TrimSpace(line[:i])
			rest := line[i+4:]
			if j := strings.Index(rest, " # "); j >= 0 {
				typ = strings.TrimSpace(rest[:j])
				label = strings.TrimSpace(rest[j+3:])
			} else {
				typ = strings.TrimSpace(rest)
			}
		}
		if dev != "" && typ != "" {
			disks = append(disks, map[string]string{"device": dev, "type": typ, "label": label})
		}
	}
	res["disks"] = disks
	return res
}

// collectSMART prefers JSON output (smartctl -j) and falls back to raw text.
// A degraded second-hand disk shows itself in counters over time, not in one read,
// so we always return the full attribute set. deviceType is optional and is
// passed as smartctl -d (e.g. "megaraid,0" for PERC H730 virtual disks).
func collectSMART(device, deviceType string) map[string]any {
	res := map[string]any{"device": device, "collected_at": time.Now().UTC().Format(time.RFC3339)}
	if deviceType != "" {
		res["device_type"] = deviceType
	}
	if !lookPath("smartctl") {
		res["error"] = "smartctl not installed"
		return res
	}
	args := []string{"-j", "-a"}
	if deviceType != "" {
		args = append(args, "-d", deviceType)
	}
	args = append(args, device)
	out, err := runCommand("smartctl", args...)
	if err == nil {
		var parsed map[string]any
		if jerr := json.Unmarshal([]byte(out), &parsed); jerr == nil {
			res["smart"] = parsed
			return res
		}
	}
	rawArgs := []string{"-a"}
	if deviceType != "" {
		rawArgs = append(rawArgs, "-d", deviceType)
	}
	rawArgs = append(rawArgs, device)
	raw, rerr := runCommand("smartctl", rawArgs...)
	res["raw"] = raw
	if rerr != nil {
		res["error"] = rerr.Error()
	}
	return res
}

// collectLogs returns bounded log excerpts. scope selects the source so the
// caller never gets a free-form shell.
func collectLogs(scope string, tail int) map[string]any {
	res := map[string]any{"scope": scope, "tail": tail, "collected_at": time.Now().UTC().Format(time.RFC3339)}
	switch scope {
	case "dmesg":
		var out string
		var err error
		if lookPath("dmesg") {
			out, err = runCommand("dmesg", "--ctime", "--color=never")
		} else if data := readFile("/var/log/dmesg"); data != "" {
			out = data
		} else {
			res["error"] = "dmesg not available"
			return res
		}
		if err != nil {
			res["error"] = err.Error()
		}
		res["lines"] = tailLines(out, tail)
	case "journal":
		if !lookPath("journalctl") {
			res["error"] = "journalctl not installed"
			return res
		}
		out, err := runCommand("journalctl", "-p", "warning", "-n", strconv.Itoa(tail), "--no-pager", "-o", "short-iso")
		if err != nil {
			res["error"] = err.Error()
		}
		res["lines"] = strings.Split(strings.TrimSpace(out), "\n")
	case "mce", "edac":
		res["mce"] = readFile("/var/log/mcelog")
		if res["mce"] == "" {
			res["mce"] = "no mcelog available"
		}
		entries, _ := os.ReadDir("/sys/devices/system/edac/mc")
		var mc []string
		for _, e := range entries {
			mc = append(mc, e.Name())
		}
		res["edac_controllers"] = mc
	default:
		res["error"] = "unknown scope: " + scope
	}
	return res
}

func tailLines(s string, n int) []string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if n > 0 && len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}
