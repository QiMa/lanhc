// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

// The lanhc command is the Lanhc command-line client. It interacts
// with the lanhcd node agent.
package main // import "lanhc.com/cmd/lanhc"

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lanhc.com/cmd/lanhc/cli"
)

func main() {
	args := os.Args[1:]
	if name, _ := os.Executable(); strings.HasSuffix(filepath.Base(name), ".cgi") {
		args = []string{"web", "-cgi"}
	}
	if err := cli.Run(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
