// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

//go:build lanhc_isolated

package lanhc

// Isolated is true in lanhc downstream builds. Code guards official Tailscale
// endpoints behind this constant so the compiler can drop the literals
// entirely; see the lanhc build script, which passes -tags lanhc_isolated.
const Isolated = true

// OfficialAdminPageURL returns an empty string in lanhc builds, where the
// official admin page is not used.
func OfficialAdminPageURL() string { return "" }
