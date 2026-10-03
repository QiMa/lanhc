// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

//go:build !windows

package main

import (
	"strconv"
	"strings"
)

// diskUsedPct returns the percentage of the filesystem containing path that is
// in use, or 0 if the value can't be determined. It shells out to df instead
// of using syscall.Statfs so the same implementation works across the BSDs,
// Linux, and Darwin without per-OS struct field differences.
func diskUsedPct(path string) float64 {
	out, err := runCommand("df", "-P", "-k", path)
	if err != nil {
		return 0
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		return 0
	}
	fields := strings.Fields(lines[len(lines)-1])
	if len(fields) < 5 {
		return 0
	}
	used, err1 := strconv.ParseFloat(fields[2], 64)
	avail, err2 := strconv.ParseFloat(fields[3], 64)
	total := used + avail
	if err1 != nil || err2 != nil || total == 0 {
		return 0
	}
	return 100 * used / total
}
