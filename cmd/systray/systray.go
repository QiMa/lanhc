// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

//go:build cgo || !darwin

// systray is a minimal Lanhc systray application.
package main

import (
	"flag"

	"lanhc.com/client/local"
	"lanhc.com/client/systray"
	"lanhc.com/paths"
)

var socket = flag.String("socket", paths.DefaultLanhcdSocket(), "path to lanhcd socket")
var theme = flag.String("theme", "dark", "color theme for Lanhc icon: dark, dark:nobg, light, light:nobg")

func main() {
	flag.Parse()
	lc := &local.Client{Socket: *socket}
	systray.SetTheme(*theme)
	new(systray.Menu).Run(lc)
}
