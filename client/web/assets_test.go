// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"lanhc.com/net/tsaddr"
	"lanhc.com/tailcfg"
)

// TestAssetsHandlerServesLocalBrandedClient is a smoke test for the web
// client assets that are built locally in client/web and embedded in the
// binary. It guards against regressions where the binary falls back to the
// external github.com/tailscale/web-client-prebuilt module or otherwise
// serves the wrong branding.
func TestAssetsHandlerServesLocalBrandedClient(t *testing.T) {
	t.Setenv("TS_DEBUG_WEB_CLIENT_DEV", "")

	s, err := NewServer(ServerOpts{
		Mode: ManageServerMode,
		NewAuthURL: func(context.Context, tailcfg.NodeID) (*tailcfg.WebClientAuthResponse, error) {
			return nil, nil
		},
		WaitAuthURL: func(context.Context, string, tailcfg.NodeID) (*tailcfg.WebClientAuthResponse, error) {
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer s.Shutdown()

	const indexScriptHash = "sha384-Qv0SDOms+LGX2fjgbkznbrQVBb/S9C6OnmI0t+SkJqJ6B7FL8p2I6WlmYbsSyhx/"

	get := func(t *testing.T, path string) (*http.Response, string) {
		t.Helper()
		r := httptest.NewRequest(http.MethodGet, "http://"+tsaddr.LanhcServiceIPString+path, nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		res := w.Result()
		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			t.Fatalf("reading %q body: %v", path, err)
		}
		return res, string(body)
	}

	res, index := get(t, "/")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /: got status %d, want %d", res.StatusCode, http.StatusOK)
	}
	if !strings.Contains(index, "<title>Lanhc</title>") {
		t.Errorf("GET /: index.html missing <title>Lanhc</title>")
	}
	if strings.Contains(index, "Tailscale") {
		t.Errorf("GET /: index.html still contains upstream branding %q", "Tailscale")
	}
	if csp := res.Header.Get("Content-Security-Policy"); !strings.Contains(csp, indexScriptHash) {
		t.Errorf("GET /: Content-Security-Policy %q does not contain script hash %q", csp, indexScriptHash)
	}

	// Every hashed asset referenced by index.html must be served from the
	// embedded filesystem.
	assetRefRE := regexp.MustCompile(`\./assets/[A-Za-z0-9_.-]+\.(js|css|woff2)`)
	refs := assetRefRE.FindAllString(index, -1)
	if len(refs) < 3 {
		t.Fatalf("GET /: found %d asset references, want at least 3: %v", len(refs), refs)
	}
	var jsRef string
	for _, ref := range refs {
		path := strings.TrimPrefix(ref, ".")
		res, _ := get(t, path)
		if res.StatusCode != http.StatusOK {
			t.Errorf("GET %s: got status %d, want %d", path, res.StatusCode, http.StatusOK)
		}
		if strings.HasSuffix(ref, ".js") {
			jsRef = path
		}
	}
	if jsRef == "" {
		t.Fatal("GET /: did not find a JavaScript asset reference")
	}

	// The JS bundle must contain the local vector icon rather than a raster
	// image from the upstream prebuilt module.
	_, js := get(t, jsRef)
	if strings.Contains(js, "data:image/png;base64") {
		t.Errorf("GET %s: bundle contains a raster PNG icon, want vector path data", jsRef)
	}
	if !strings.Contains(js, "M 0 0 L 64 0 L 64 64 L 0 64") {
		t.Errorf("GET %s: bundle is missing the local vector icon path data", jsRef)
	}

	// Unknown client-side routes must fall back to index.html.
	res, body := get(t, "/login/some-client-route")
	if res.StatusCode != http.StatusOK {
		t.Errorf("GET /login/some-client-route: got status %d, want %d", res.StatusCode, http.StatusOK)
	}
	if !strings.Contains(body, "<title>Lanhc</title>") {
		t.Errorf("GET /login/some-client-route: fallback response missing <title>Lanhc</title>")
	}
}
