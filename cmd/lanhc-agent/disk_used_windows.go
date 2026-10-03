// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

//go:build windows

package main

import "golang.org/x/sys/windows"

// diskUsedPct returns the percentage of the filesystem containing path that is
// in use, or 0 if the value can't be determined.
func diskUsedPct(path string) float64 {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0
	}
	var freeBytesAvailable, totalBytes, totalFreeBytes uint64
	if err := windows.GetDiskFreeSpaceEx(p, &freeBytesAvailable, &totalBytes, &totalFreeBytes); err != nil {
		return 0
	}
	if totalBytes == 0 {
		return 0
	}
	return 100 * float64(totalBytes-totalFreeBytes) / float64(totalBytes)
}
