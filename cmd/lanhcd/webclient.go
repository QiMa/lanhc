// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

//go:build !ts_omit_webclient

package main

import (
	"lanhc.com/client/local"
	"lanhc.com/ipn/ipnlocal"
	"lanhc.com/paths"
)

func init() {
	hookConfigureWebClient.Set(func(lb *ipnlocal.LocalBackend) {
		lb.ConfigureWebClient(&local.Client{
			Socket:        args.socketpath,
			UseSocketOnly: args.socketpath != paths.DefaultLanhcdSocket(),
		})
	})
}
