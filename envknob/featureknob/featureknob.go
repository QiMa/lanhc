// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

// Package featureknob provides a facility to control whether features
// can run based on either an envknob or running OS / distro.
package featureknob

import (
	"errors"
	"runtime"

	"lanhc.com/envknob"
	"lanhc.com/version"
	"lanhc.com/version/distro"
)

// CanRunLanhcSSH reports whether serving a Lanhc SSH server is
// supported for the current os/distro.
func CanRunLanhcSSH() error {
	switch runtime.GOOS {
	case "linux":
		if distro.Get() == distro.Synology && !envknob.UseWIPCode() {
			return errors.New("The Lanhc SSH server does not run on Synology.")
		}
		if distro.Get() == distro.QNAP && !envknob.UseWIPCode() {
			return errors.New("The Lanhc SSH server does not run on QNAP.")
		}
		// otherwise okay
	case "darwin":
		// okay only in lanhcd mode for now.
		if version.IsSandboxedMacOS() {
			return errors.New("The Lanhc SSH server does not run in sandboxed Lanhc GUI builds.")
		}
	case "freebsd", "openbsd", "plan9":
	default:
		return errors.New("The Lanhc SSH server is not supported on " + runtime.GOOS)
	}
	if !envknob.CanSSHD() {
		return errors.New("The Lanhc SSH server has been administratively disabled.")
	}
	return nil
}

// CanUseExitNode reports whether using an exit node is supported for the
// current os/distro.
func CanUseExitNode() error {
	switch dist := distro.Get(); dist {
	case distro.Synology, // see https://github.com/lanhc/lanhc/issues/1995
		distro.QNAP:
		return errors.New("Lanhc exit nodes cannot be used on " + string(dist))
	}
	return nil
}
