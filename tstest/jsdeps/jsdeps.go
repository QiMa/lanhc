// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

// Package jsdeps is a just a list of the packages we import in the
// JavaScript/WASM build, to let us test that our transitive closure of
// dependencies doesn't accidentally grow too large, since binary size
// is more of a concern.
package jsdeps

import (
	_ "bytes"
	_ "context"
	_ "encoding/hex"
	_ "encoding/json"
	_ "fmt"
	_ "log"
	_ "math/rand/v2"
	_ "net"
	_ "strings"
	_ "time"

	_ "golang.org/x/crypto/ssh"
	_ "lanhc.com/control/controlclient"
	_ "lanhc.com/ipn"
	_ "lanhc.com/ipn/ipnserver"
	_ "lanhc.com/net/netaddr"
	_ "lanhc.com/net/netns"
	_ "lanhc.com/net/tsdial"
	_ "lanhc.com/safesocket"
	_ "lanhc.com/tailcfg"
	_ "lanhc.com/types/logger"
	_ "lanhc.com/wgengine"
	_ "lanhc.com/wgengine/netstack"
	_ "lanhc.com/words"
)
