// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

// Package lanhcdhooks provides hooks for optional features
// to add to during init that lanhcd calls at runtime.
package lanhcdhooks

import "lanhc.com/feature"

// UninstallSystemDaemonWindows is called when the Windows
// system daemon is uninstalled.
var UninstallSystemDaemonWindows feature.Hooks[func()]
