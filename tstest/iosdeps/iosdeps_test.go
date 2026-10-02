// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package iosdeps

import (
	"testing"

	"lanhc.com/tstest/deptest"
)

func TestDeps(t *testing.T) {
	deptest.DepChecker{
		GOOS:   "ios",
		GOARCH: "arm64",
		BadDeps: map[string]string{
			"testing":                         "do not use testing package in production code",
			"text/template":                   "linker bloat (MethodByName)",
			"html/template":                   "linker bloat (MethodByName)",
			"lanhc.com/net/wsconn":            "https://github.com/lanhc/lanhc/issues/13762",
			"github.com/coder/websocket":      "https://github.com/lanhc/lanhc/issues/13762",
			"github.com/mitchellh/go-ps":      "https://github.com/lanhc/lanhc/pull/13759",
			"database/sql/driver":             "iOS doesn't use an SQL database",
			"github.com/google/uuid":          "see lanhc/lanhc#13760",
			"lanhc.com/clientupdate/distsign": "downloads via AppStore, not distsign",
			"github.com/tailscale/hujson":     "no config file support on iOS",
			"lanhc.com/feature/capture":       "no debug packet capture on iOS",
		},
	}.Check(t)
}
