// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

//go:build !ts_omit_routecheck

package ipnlocal

import (
	"lanhc.com/net/routecheck"
	"lanhc.com/tailcfg"
)

func isRouteCheckEnabled(self tailcfg.NodeView) bool {
	return routecheck.IsEnabled(self)
}
