// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package cli

import (
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
)

var dnsCmd = &ffcli.Command{
	Name:      "dns",
	ShortHelp: "Diagnose the internal DNS forwarder",
	LongHelp: strings.TrimSpace(`
The 'lanhc dns' subcommand provides tools for diagnosing the internal DNS
forwarder (100.100.100.100).

For more information about the DNS functionality built into Lanhc, refer to
https://lanhc.com/kb/1054/dns.
`),
	ShortUsage: strings.Join([]string{
		dnsStatusCmd.ShortUsage,
		dnsQueryCmd.ShortUsage,
	}, "\n"),
	UsageFunc: usageFuncNoDefaultValues,
	Subcommands: []*ffcli.Command{
		dnsStatusCmd,
		dnsQueryCmd,

		// TODO: implement `lanhc log` here

		// The above work is tracked in https://github.com/lanhc/lanhc/issues/13326
	},
}
