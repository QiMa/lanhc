// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

//go:build !lanhc_isolated

// Package lanhc holds compile-time switches for lanhc downstream builds.
package lanhc

// Isolated is false in upstream builds. In this mode the client keeps its
// historical behaviour, including any official Lanhc endpoints that were
// not injected at link time.
const Isolated = false

// OfficialAdminPageURL is the upstream admin page URL.
func OfficialAdminPageURL() string { return "https://login.lanhc.com/admin" }
