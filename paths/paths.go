// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

// Package paths returns platform and user-specific default paths to
// Lanhc files and directories.
package paths

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"lanhc.com/syncs"
	"lanhc.com/version/distro"
)

// AppSharedDir is a string set by the iOS or Android app on start
// containing a directory we can read/write in.
var AppSharedDir syncs.AtomicValue[string]

// DefaultLanhcdSocket returns the path to the lanhcd Unix socket
// or the empty string if there's no reasonable default.
func DefaultLanhcdSocket() string {
	if runtime.GOOS == "windows" {
		return `\\.\pipe\ProtectedPrefix\Administrators\Lanhc\lanhcd`
	}
	if runtime.GOOS == "darwin" {
		return "/var/run/lanhcd.socket"
	}
	if runtime.GOOS == "plan9" {
		return "/srv/lanhcd.sock"
	}
	switch distro.Get() {
	case distro.Synology:
		if distro.DSMVersion() == 6 {
			return "/var/packages/Lanhc/etc/lanhcd.sock"
		}
		// DSM 7 (and higher? or failure to detect.)
		return "/var/packages/Lanhc/var/lanhcd.sock"
	case distro.Gokrazy:
		return "/perm/lanhcd/lanhcd.sock"
	case distro.QNAP:
		return "/tmp/lanhc/lanhcd.sock"
	}
	if fi, err := os.Stat("/var/run"); err == nil && fi.IsDir() {
		return "/var/run/lanhc/lanhcd.sock"
	}
	return "lanhcd.sock"
}

// Overridden in init by OS-specific files.
var (
	stateFileFunc func() string

	// ensureStateDirPerms applies a restrictive ACL/chmod
	// to the provided directory.
	ensureStateDirPerms = func(string) error { return nil }
)

// DefaultLanhcdStateFile returns the default path to the
// lanhcd state file, or the empty string if there's no reasonable
// default value.
func DefaultLanhcdStateFile() string {
	if f := stateFileFunc; f != nil {
		return f()
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("ProgramData"), "Lanhc", "server-state.conf")
	}
	return ""
}

// DefaultLanhcdStateDir returns the default state directory
// to use for lanhcd, for use when the user provided neither
// a state directory or state file path to use.
//
// It returns the empty string if there's no reasonable default.
func DefaultLanhcdStateDir() string {
	if runtime.GOOS == "plan9" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("failed to get home directory: %v", err)
		}
		return filepath.Join(home, "lanhc-state")
	}
	return filepath.Dir(DefaultLanhcdStateFile())
}

// MakeAutomaticStateDir reports whether the platform
// automatically creates the state directory for lanhcd
// when it's absent.
func MakeAutomaticStateDir() bool {
	switch runtime.GOOS {
	case "plan9":
		return true
	case "linux":
		if distro.Get() == distro.JetKVM {
			return true
		}
	}
	return false
}

// MkStateDir ensures that dirPath, the daemon's configuration directory
// containing machine keys etc, both exists and has the correct permissions.
// We want it to only be accessible to the user the daemon is running under.
func MkStateDir(dirPath string) error {
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return err
	}
	return ensureStateDirPerms(dirPath)
}

// LegacyStateFilePath returns the legacy path to the state file when
// it was stored under the current user's %LocalAppData%.
//
// It is only called on Windows.
func LegacyStateFilePath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("LocalAppData"), "Lanhc", "server-state.conf")
	}
	return ""
}
