// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"testing"

	"lanhc.com/tstest/deptest"
)

func TestDeps(t *testing.T) {
	deptest.DepChecker{
		BadDeps: map[string]string{
			"lanhc.com/tailcfg": "circular dependency via go generate",
			"lanhc.com/version": "circular dependency via go generate",
		},
	}.Check(t)
}
