// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"net"
	"net/netip"
	"strings"
	"testing"

	"lanhc.com/net/memnet"
)

func echoConnOnce(conn net.Conn) {
	defer conn.Close()

	b := make([]byte, 256)
	n, err := conn.Read(b)
	if err != nil {
		return
	}

	if _, err := conn.Write(b[:n]); err != nil {
		return
	}
}

func TestTCPRoundRobinHandler(t *testing.T) {
	h := tcpRoundRobinHandler{
		To: []string{"yeet.com"},
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if network != "tcp" {
				t.Errorf("network = %s, want %s", network, "tcp")
			}
			if addr != "yeet.com:22" {
				t.Errorf("addr = %s, want %s", addr, "yeet.com:22")
			}

			c, s := memnet.NewConn("outbound", 1024)
			go echoConnOnce(s)
			return c, nil
		},
	}

	cSock, sSock := memnet.NewTCPConn(netip.MustParseAddrPort("10.64.1.2:22"), netip.MustParseAddrPort("10.64.1.2:22"), 1024)
	h.Handle(sSock)

	// Test data write and read, the other end will echo back
	// a single stanza
	want := "hello"
	if _, err := io.WriteString(cSock, want); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len(want))
	if _, err := io.ReadAtLeast(cSock, got, len(got)); err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// The other end closed the socket after the first echo, so
	// any following read should error.
	io.WriteString(cSock, "deadass heres some data on god fr")
	if _, err := io.ReadAtLeast(cSock, got, len(got)); err == nil {
		t.Error("read succeeded on closed socket")
	}
}

// Capture of first TCP data segment for a connection to https://pkgs.lanhc.com
const tlsStart = `45000239ff1840004006f9f5c0a801f2
c726b5efcf9e01bbe803b21394e3b752
801801f641dc00000101080ade3474f2
2fb93ee716030100e1010000dd030331
4764b50a6c59ce32e6666393b9272e74
d56ae76396a00dee5e06608482469c20
65a8cd97e0586aaa9b00e65bdd85cd0b
3985ba2979f582dbb72e9d9aaf316908
0014c02bc02fc02cc030cca9cca8c009
c013c00ac01401000080000000130011
00000e706b67732e6c616e68632e636f
6d000b00020100ff0100010000170000
00120000000500050100000000000a00
0a0008001d001700180019000d001600
14080404030807080508060401050106
01050306030032001a00180804040308
07080508060401050106010503060302
010203002b0003020303`

func fakeSNIHeader() []byte {
	b, err := hex.DecodeString(strings.Replace(tlsStart, "\n", "", -1))
	if err != nil {
		panic(err)
	}
	return b[0x34:] // trim IP + TCP header
}

func TestTCPSNIHandler(t *testing.T) {
	h := tcpSNIHandler{
		Allowlist: []string{"pkgs.lanhc.com"},
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if network != "tcp" {
				t.Errorf("network = %s, want %s", network, "tcp")
			}
			if addr != "pkgs.lanhc.com:443" {
				t.Errorf("addr = %s, want %s", addr, "pkgs.lanhc.com:443")
			}

			c, s := memnet.NewConn("outbound", 1024)
			go echoConnOnce(s)
			return c, nil
		},
	}

	cSock, sSock := memnet.NewTCPConn(netip.MustParseAddrPort("10.64.1.2:22"), netip.MustParseAddrPort("10.64.1.2:443"), 1024)
	h.Handle(sSock)

	// Fake a TLS handshake record with an SNI in it.
	if _, err := cSock.Write(fakeSNIHeader()); err != nil {
		t.Fatal(err)
	}

	// Test read, the other end will echo back
	// a single stanza, which is at least the beginning of the SNI header.
	want := fakeSNIHeader()[:5]
	if _, err := cSock.Write(want); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len(want))
	if _, err := io.ReadAtLeast(cSock, got, len(got)); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}
