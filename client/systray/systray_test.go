// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

//go:build cgo || !darwin

package systray

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"lanhc.com/ipn/ipnstate"
	"lanhc.com/tailcfg"
	"lanhc.com/types/key"
)

func TestProfileTitleMultiline(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		login     string
		tailnet   string
		multiline bool
		want      string
	}{
		{"no_tailnet", "alice@example.com", "", true, "alice@example.com"},
		{"dup_exact", "example.com", "example.com", true, "example.com"},
		{"dup_casefold", "Example.com", "example.com", false, "Example.com"},
		{"distinct_multiline", "alice@example.com", "example.com", true, "alice@example.com\nexample.com"},
		{"distinct_singleline", "alice@example.com", "example.com", false, "alice@example.com (example.com)"},
		{"empty", "", "", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := formatProfileTitle(tt.login, tt.tailnet, tt.multiline); got != tt.want {
				t.Errorf("profileTitleMultiline; got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRecommendedIsActive(t *testing.T) {
	t.Parallel()

	const (
		activeID = tailcfg.StableNodeID("active")
		suggID   = tailcfg.StableNodeID("suggestion")
	)
	usNYC := &tailcfg.Location{CountryCode: "US", City: "New York"}
	usCHI := &tailcfg.Location{CountryCode: "US", City: "Chicago"}
	seSTO := &tailcfg.Location{CountryCode: "SE", City: "Stockholm"}

	statusWith := func(activePeer *ipnstate.PeerStatus) *ipnstate.Status {
		s := &ipnstate.Status{
			ExitNodeStatus: &ipnstate.ExitNodeStatus{ID: activeID},
		}
		if activePeer != nil {
			s.Peer = map[key.NodePublic]*ipnstate.PeerStatus{{}: activePeer}
		}
		return s
	}

	tests := []struct {
		name        string
		status      *ipnstate.Status
		suggID      tailcfg.StableNodeID
		suggCountry string
		suggCity    string
		isActive    bool
	}{
		{
			name:   "nil_status",
			status: nil,
			suggID: suggID,
		},
		{
			name:   "no_exit_node",
			status: &ipnstate.Status{},
			suggID: suggID,
		},
		{
			name:   "exit_node_id_is_zero",
			status: &ipnstate.Status{ExitNodeStatus: &ipnstate.ExitNodeStatus{}},
			suggID: suggID,
		},
		{
			name:        "exact_id_match_short-circuits",
			status:      statusWith(&ipnstate.PeerStatus{ID: activeID, Location: usCHI}),
			suggID:      activeID,
			suggCountry: "US",
			suggCity:    "New York",
			isActive:    true,
		},
		{
			name:        "id_mismatch_but_same_city",
			status:      statusWith(&ipnstate.PeerStatus{ID: activeID, Location: usNYC}),
			suggID:      suggID,
			suggCountry: "US",
			suggCity:    "New York",
			isActive:    true,
		},
		{
			name:        "different_city",
			status:      statusWith(&ipnstate.PeerStatus{ID: activeID, Location: usCHI}),
			suggID:      suggID,
			suggCountry: "US",
			suggCity:    "New York",
		},
		{
			name:        "different_country",
			status:      statusWith(&ipnstate.PeerStatus{ID: activeID, Location: seSTO}),
			suggID:      suggID,
			suggCountry: "US",
			suggCity:    "New York",
		},
		{
			name:   "id_mismatch_suggestion_has_no_location",
			status: statusWith(&ipnstate.PeerStatus{ID: activeID, Location: usNYC}),
			suggID: suggID,
		},
		{
			name:        "id_mismatch_active_peer_has_no_location",
			status:      statusWith(&ipnstate.PeerStatus{ID: activeID}),
			suggID:      suggID,
			suggCountry: "US",
			suggCity:    "New York",
		},
		{
			name:        "active_peer_not_in_status",
			status:      statusWith(nil),
			suggID:      suggID,
			suggCountry: "US",
			suggCity:    "New York",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			isExitNodeActive := recommendedIsActive(tt.status, tt.suggID, tt.suggCountry, tt.suggCity)
			if isExitNodeActive != tt.isActive {
				t.Errorf("recommendedIsActive; got %v, want %v", isExitNodeActive, tt.isActive)
			}
		})
	}
}

func TestBrandIconStates(t *testing.T) {
	t.Parallel()

	conn := brandConnected.render()
	disc := brandDisconnected.render()
	if conn.Len() == 0 || disc.Len() == 0 {
		t.Fatal("brand icons rendered empty")
	}

	connImg, err := png.Decode(bytes.NewReader(conn.Bytes()))
	if err != nil {
		t.Fatalf("decoding connected icon: %v", err)
	}
	discImg, err := png.Decode(bytes.NewReader(disc.Bytes()))
	if err != nil {
		t.Fatalf("decoding disconnected icon: %v", err)
	}

	// Connected must keep the brand colors; disconnected must be grayscale.
	// Compare the same pixel in both to make sure they are actually different.
	b := connImg.Bounds()
	colored, gray := 0, 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := connImg.At(x, y).RGBA(); a == 0 {
				continue
			}
			r, g, bl, _ := connImg.At(x, y).RGBA()
			if r>>8 != g>>8 || g>>8 != bl>>8 {
				colored++
			}
			r, g, bl, _ = discImg.At(x, y).RGBA()
			if r == g && g == bl {
				gray++
			}
		}
	}
	if colored == 0 {
		t.Error("connected icon has no colored pixels")
	}
	if gray == 0 {
		t.Error("disconnected icon has no grayscale pixels")
	}

	// The brand mark must sit on a transparent canvas so it blends into the
	// taskbar instead of showing a black square.
	for _, corner := range []image.Point{
		connImg.Bounds().Min,
		{X: connImg.Bounds().Max.X - 1, Y: connImg.Bounds().Min.Y},
		{X: connImg.Bounds().Min.X, Y: connImg.Bounds().Max.Y - 1},
		{X: connImg.Bounds().Max.X - 1, Y: connImg.Bounds().Max.Y - 1},
	} {
		if _, _, _, a := connImg.At(corner.X, corner.Y).RGBA(); a != 0 {
			t.Errorf("connected icon corner %v is not transparent: alpha=%d", corner, a)
		}
		if _, _, _, a := discImg.At(corner.X, corner.Y).RGBA(); a != 0 {
			t.Errorf("disconnected icon corner %v is not transparent: alpha=%d", corner, a)
		}
	}
}
