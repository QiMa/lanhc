// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package integration

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/proxy"
	"lanhc.com/tstest"
	"lanhc.com/util/httpm"
)

// TestWebClientOverNetstack is an end-to-end test of the management web client
// exposed over Lanhc. It brings up a node with userspace networking, enables the
// remote web client, and then fetches the client through the node's own netstack
// (via its SOCKS5 proxy) at http://<nodeIP>:5252, verifying that the locally
// built and embedded assets are served with the expected branding and gzip
// precompression.
func TestWebClientOverNetstack(t *testing.T) {
	tstest.Parallel(t)
	env := NewTestEnv(t)
	n := NewTestNode(t, env)
	socksCh := n.socks5AddrChan()
	d := n.StartDaemon()
	defer d.MustCleanShutdown(t)

	n.AwaitResponding()
	n.MustUp()
	n.AwaitRunning()
	socksAddr := n.AwaitSocksAddr(socksCh)
	nodeIP := n.AwaitIP4()

	if err := n.Lanhc("set", "--webclient=true").Run(); err != nil {
		t.Fatalf("enabling web client: %v", err)
	}

	dialer, err := proxy.SOCKS5("tcp", socksAddr, nil, &net.Dialer{})
	if err != nil {
		t.Fatalf("creating SOCKS5 dialer: %v", err)
	}
	cd, ok := dialer.(proxy.ContextDialer)
	if !ok {
		t.Fatalf("SOCKS5 dialer %T does not implement proxy.ContextDialer", dialer)
	}
	client := &http.Client{
		Transport: &http.Transport{
			DialContext:        cd.DialContext,
			DisableCompression: true, // we want to see the raw Content-Encoding
		},
	}
	baseURL := "http://" + net.JoinHostPort(nodeIP.String(), "5252")

	get := func(t *testing.T, path, acceptEncoding string) (*http.Response, []byte) {
		t.Helper()
		var (
			resBody []byte
			res     *http.Response
		)
		if err := tstest.WaitFor(20*time.Second, func() error {
			req, err := http.NewRequest(httpm.GET, baseURL+path, nil)
			if err != nil {
				return err
			}
			if acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", acceptEncoding)
			}
			r, err := client.Do(req)
			if err != nil {
				return err
			}
			defer r.Body.Close()
			resBody, err = io.ReadAll(r.Body)
			if err != nil {
				return err
			}
			if r.StatusCode != http.StatusOK {
				return fmt.Errorf("GET %s: status %d", path, r.StatusCode)
			}
			res = r
			return nil
		}); err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		return res, resBody
	}

	res, index := get(t, "/", "")
	if !strings.Contains(string(index), "<title>Lanhc</title>") {
		t.Errorf("GET /: index.html missing <title>Lanhc</title>")
	}
	if strings.Contains(string(index), "Tailscale") {
		t.Errorf("GET /: index.html still contains upstream branding")
	}
	const indexScriptHash = "sha384-Qv0SDOms+LGX2fjgbkznbrQVBb/S9C6OnmI0t+SkJqJ6B7FL8p2I6WlmYbsSyhx/"
	if csp := res.Header.Get("Content-Security-Policy"); !strings.Contains(csp, indexScriptHash) {
		t.Errorf("GET /: Content-Security-Policy %q does not contain script hash %q", csp, indexScriptHash)
	}

	// Fetch the JS bundle advertised by index.html with gzip and confirm it is
	// served precompressed over the embedded filesystem.
	jsMatch := regexp.MustCompile(`\./assets/[A-Za-z0-9_.-]+\.js`).FindString(string(index))
	if jsMatch == "" {
		t.Fatalf("GET /: no JS asset reference found in index.html")
	}
	jsPath := strings.TrimPrefix(jsMatch, ".")
	res, compressed := get(t, jsPath, "gzip")
	if got := res.Header.Get("Content-Encoding"); got != "gzip" {
		t.Errorf("GET %s with Accept-Encoding: gzip: got Content-Encoding %q, want gzip", jsPath, got)
	}
	zr, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("gzip.NewReader(%s): %v", jsPath, err)
	}
	decompressed, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("decompressing %s: %v", jsPath, err)
	}
	zr.Close()
	js := string(decompressed)
	if strings.Contains(js, "data:image/png;base64") {
		t.Errorf("GET %s: bundle contains a raster PNG icon, want vector path data", jsPath)
	}
	if !strings.Contains(js, "M 0 0 L 64 0 L 64 64 L 0 64") {
		t.Errorf("GET %s: bundle is missing the local vector icon path data", jsPath)
	}

	// Unknown client-side routes fall back to index.html.
	_, body := get(t, "/login/some-client-route", "")
	if !strings.Contains(string(body), "<title>Lanhc</title>") {
		t.Errorf("GET /login/some-client-route: fallback response missing <title>Lanhc</title>")
	}
}
