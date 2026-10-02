// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

// The web-client command demonstrates serving the Lanhc web client over tsnet.
package main

import (
	"flag"
	"log"
	"net/http"

	"lanhc.com/client/web"
	"lanhc.com/tsnet"
)

var (
	addr = flag.String("addr", "localhost:8060", "address of Lanhc web client")
)

func main() {
	flag.Parse()

	s := &tsnet.Server{RunWebClient: true}
	defer s.Close()

	lc, err := s.LocalClient()
	if err != nil {
		log.Fatal(err)
	}

	// Serve the Lanhc web client.
	ws, err := web.NewServer(web.ServerOpts{
		Mode:        web.LoginServerMode,
		LocalClient: lc,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Shutdown()
	log.Printf("Serving Lanhc web client on http://%s", *addr)
	if err := http.ListenAndServe(*addr, ws); err != nil {
		if err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}
}
