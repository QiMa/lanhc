// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

// Package vet is a tool to statically check Go source code.
package main

import (
	_ "embed"

	"golang.org/x/tools/go/analysis/unitchecker"
	"lanhc.com/cmd/vet/jsontags"
	"lanhc.com/cmd/vet/lowerell"
	"lanhc.com/cmd/vet/subtestnames"
)

//go:embed jsontags_allowlist
var jsontagsAllowlistSource string

func init() {
	jsontags.RegisterAllowlist(jsontags.ParseAllowlist(jsontagsAllowlistSource))
	jsontags.RegisterPureIsZeroMethods(jsontags.PureIsZeroMethodsInLanhcModule)
}

func main() {
	unitchecker.Main(jsontags.Analyzer, lowerell.Analyzer, subtestnames.Analyzer)
}
