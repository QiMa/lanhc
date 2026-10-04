// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

// Command lanhc-agent is the device-side companion to lanhc. It joins the same
// tailnet via tsnet and exposes a small, read-only HTTP API so ops-runner can
// collect evidence without SSH or a free-form shell.
//
// It is a separate binary on purpose: lanhcd stays untouched and the agent can
// be upgraded or disabled independently. It never holds Codex or LLM credentials.
//
// Usage:
//
//	TS_AUTHKEY=<tagged-preauthkey> lanhc-agent -dir /var/lib/lanhc-agent
//	lanhc-agent -selfcheck            # collect local data without joining a tailnet
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"lanhc.com/ipn"
	"lanhc.com/tsnet"
)

var version = "0.1.0"

func main() {
	var (
		hostname  = flag.String("hostname", envOr("LANHC_AGENT_HOSTNAME", defaultHostname()), "tailnet hostname to register")
		dir       = flag.String("dir", envOr("LANHC_AGENT_DIR", "/var/lib/lanhc-agent"), "state directory for the embedded node")
		control   = flag.String("control-url", envOr("LANHC_AGENT_CONTROL_URL", ""), "control server URL (defaults to the compiled-in lanhc control plane)")
		authKey   = flag.String("auth-key", "", "auth key for first enrollment (reads TS_AUTHKEY too)")
		listen    = flag.String("listen", envOr("LANHC_AGENT_LISTEN", ":8088"), "tailnet address to listen on")
		advertise = flag.String("tags", envOr("LANHC_AGENT_TAGS", ""), "comma-separated ACL tags to request (usually leave empty; tags come from the preauthkey)")
		selfCheck = flag.Bool("selfcheck", false, "collect and print local data, then exit (no tailnet)")
	)
	flag.Parse()

	if *selfCheck {
		runSelfCheck()
		return
	}

	controlURL := strings.TrimSpace(*control)
	if controlURL == "" {
		controlURL = ipn.DefaultControlURL
	}

	s := &tsnet.Server{
		Hostname:   *hostname,
		Dir:        *dir,
		ControlURL: controlURL,
		AuthKey:    *authKey,
		Logf:       func(format string, args ...any) { log.Printf("[tsnet] "+format, args...) },
	}
	if *advertise != "" {
		s.AdvertiseTags = splitTags(*advertise)
	}
	defer s.Close()

	ln, err := s.Listen("tcp", *listen)
	if err != nil {
		log.Fatalf("lanhc-agent: failed to listen on tailnet %s: %v", *listen, err)
	}

	srv := &http.Server{
		Handler:           newMux(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("lanhc-agent %s listening on tailnet %s (hostname=%s)", version, *listen, *hostname)
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		log.Fatalf("lanhc-agent: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func defaultHostname() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		return "lanhc-agent"
	}
	host = strings.TrimSuffix(host, ".local")
	if strings.HasSuffix(host, "-agent") {
		return host
	}
	return host + "-agent"
}

func splitTags(s string) []string {
	var out []string
	for _, t := range splitComma(s) {
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func splitComma(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ',' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	out = append(out, cur)
	return out
}

// runSelfCheck exercises the collectors without joining a tailnet, which is how
// the agent is smoke-tested in CI and on a fresh host.
func runSelfCheck() {
	inv := collectInventory()
	health := collectHealth()
	fmt.Printf("lanhc-agent %s selfcheck\n", version)
	fmt.Printf("  hostname=%s os=%s arch=%s cpus=%d mem=%dMB uptime=%.0fs\n",
		inv.Hostname, inv.OS, inv.Arch, inv.CPUCount, inv.MemTotalKB/1024, inv.UptimeSec)
	fmt.Printf("  disks=%d failed_units=%d root_used=%.1f%% mem_used=%.1f%%\n",
		len(inv.Disks), len(health.FailedUnits), health.RootUsedPct, health.MemUsedPct)
	gpus := collectGPUs()
	switch {
	case gpus.Error != "" && len(gpus.GPUs) == 0:
		fmt.Printf("  gpus=0 (%s)\n", gpus.Error)
	default:
		fmt.Printf("  gpus=%d", len(gpus.GPUs))
		for _, g := range gpus.GPUs {
			fmt.Printf(" [%d %s %.0f%% %.0f\u00b0C]", g.Index, g.Name, g.UtilPct, g.TemperatureC)
		}
		fmt.Println()
	}
	if len(health.FailedUnits) > 0 {
		fmt.Printf("  failed: %v\n", health.FailedUnits)
	}
	if _, err := os.Stat("/sys/block"); err != nil {
		fmt.Printf("  note: /sys/block unavailable: %v\n", err)
	}
}
