// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package lanhc

import (
	"context"

	"lanhc.com/client/local"
	"lanhc.com/client/lanhc/apitype"
	"lanhc.com/ipn/ipnstate"
)

// ErrPeerNotFound is an alias for [lanhc.com/client/local.ErrPeerNotFound].
//
// Deprecated: import [lanhc.com/client/local] instead.
var ErrPeerNotFound = local.ErrPeerNotFound

// LocalClient is an alias for [lanhc.com/client/local.Client].
//
// Deprecated: import [lanhc.com/client/local] instead.
type LocalClient = local.Client

// IPNBusWatcher is an alias for [lanhc.com/client/local.IPNBusWatcher].
//
// Deprecated: import [lanhc.com/client/local] instead.
type IPNBusWatcher = local.IPNBusWatcher

// BugReportOpts is an alias for [lanhc.com/client/local.BugReportOpts].
//
// Deprecated: import [lanhc.com/client/local] instead.
type BugReportOpts = local.BugReportOpts

// PingOpts is an alias for [lanhc.com/client/local.PingOpts].
//
// Deprecated: import [lanhc.com/client/local] instead.
type PingOpts = local.PingOpts

// SetVersionMismatchHandler is an alias for [lanhc.com/client/local.SetVersionMismatchHandler].
//
// Deprecated: import [lanhc.com/client/local] instead.
func SetVersionMismatchHandler(f func(clientVer, serverVer string)) {
	local.SetVersionMismatchHandler(f)
}

// IsAccessDeniedError is an alias for [lanhc.com/client/local.IsAccessDeniedError].
//
// Deprecated: import [lanhc.com/client/local] instead.
func IsAccessDeniedError(err error) bool {
	return local.IsAccessDeniedError(err)
}

// IsPreconditionsFailedError is an alias for [lanhc.com/client/local.IsPreconditionsFailedError].
//
// Deprecated: import [lanhc.com/client/local] instead.
func IsPreconditionsFailedError(err error) bool {
	return local.IsPreconditionsFailedError(err)
}

// WhoIs is an alias for [lanhc.com/client/local.WhoIs].
//
// Deprecated: import [lanhc.com/client/local] instead and use [local.Client.WhoIs].
func WhoIs(ctx context.Context, remoteAddr string) (*apitype.WhoIsResponse, error) {
	return local.WhoIs(ctx, remoteAddr)
}

// Status is an alias for [lanhc.com/client/local.Status].
//
// Deprecated: import [lanhc.com/client/local] instead.
func Status(ctx context.Context) (*ipnstate.Status, error) {
	return local.Status(ctx)
}

// StatusWithoutPeers is an alias for [lanhc.com/client/local.StatusWithoutPeers].
//
// Deprecated: import [lanhc.com/client/local] instead.
func StatusWithoutPeers(ctx context.Context) (*ipnstate.Status, error) {
	return local.StatusWithoutPeers(ctx)
}
