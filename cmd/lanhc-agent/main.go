// Command lanhc-agent is the device-side companion to lanhc. It joins the same
// tailnet via tsnet and exposes a small, read-only HTTP API so ops-runner can
// collect evidence without SSH or a free-form shell.
//
// It is a separate binary on purpose: lanhcd stays untouched and the agent can
// be upgraded or disabled independently. It never holds Codex or LLM credentials.
//
// Usage:
//
//	lanhc-agent -hostname r930-01-agent -dir /var/lib/lanhc-agent -listen :8088
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
	"syscall"
	"time"

	"lanhc.com/tsnet"
)

var version = "0.1.0"

func main() {
	var (
		hostname  = flag.String("hostname", "lanhc-agent", "tailnet hostname to register")
		dir       = flag.String("dir", "/var/lib/lanhc-agent", "state directory for the embedded node")
		control   = flag.String("control-url", "", "control server URL (defaults to the client's built-in lanhc control plane)")
		authKey   = flag.String("auth-key", "", "auth key for first enrollment (optional if already enrolled)")
		listen    = flag.String("listen", ":8088", "tailnet address to listen on")
		advertise = flag.String("tags", "tag:lanhc-agent", "comma-separated ACL tags to advertise")
		selfCheck = flag.Bool("selfcheck", false, "collect and print local data, then exit (no tailnet)")
	)
	flag.Parse()

	if *selfCheck {
		runSelfCheck()
		return
	}

	s := &tsnet.Server{
		Hostname:   *hostname,
		Dir:        *dir,
		ControlURL: *control,
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
	if len(health.FailedUnits) > 0 {
		fmt.Printf("  failed: %v\n", health.FailedUnits)
	}
	if _, err := os.Stat("/sys/block"); err != nil {
		fmt.Printf("  note: /sys/block unavailable: %v\n", err)
	}
}
