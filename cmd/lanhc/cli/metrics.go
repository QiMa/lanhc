// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"lanhc.com/atomicfile"
)

var metricsCmd = &ffcli.Command{
	Name:      "metrics",
	ShortHelp: "Show Lanhc metrics",
	LongHelp: strings.TrimSpace(`

The 'lanhc metrics' command shows Lanhc user-facing metrics (as opposed
to internal metrics printed by 'lanhc debug metrics').

For more information about Lanhc metrics, refer to
https://lanhc.com/s/client-metrics

`),
	ShortUsage: "lanhc metrics <subcommand> [flags]",
	UsageFunc:  usageFuncNoDefaultValues,
	Exec:       runMetricsNoSubcommand,
	Subcommands: []*ffcli.Command{
		{
			Name:       "print",
			ShortUsage: "lanhc metrics print",
			Exec:       runMetricsPrint,
			ShortHelp:  "Print current metric values in Prometheus text format",
		},
		{
			Name:       "write",
			ShortUsage: "lanhc metrics write <path>",
			Exec:       runMetricsWrite,
			ShortHelp:  "Write metric values to a file",
			LongHelp: strings.TrimSpace(`

The 'lanhc metrics write' command writes metric values to a text file provided as its
only argument. It's meant to be used alongside Prometheus node exporter, allowing Lanhc
metrics to be consumed and exported by the textfile collector.

As an example, to export Lanhc metrics on an Ubuntu system running node exporter, you
can regularly run 'lanhc metrics write /var/lib/prometheus/node-exporter/lanhcd.prom'
using cron or a systemd timer.

	`),
		},
	},
}

// runMetricsNoSubcommand prints metric values if no subcommand is specified.
func runMetricsNoSubcommand(ctx context.Context, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("lanhc metrics: unknown subcommand: %s", args[0])
	}

	return runMetricsPrint(ctx, args)
}

// runMetricsPrint prints metric values to stdout.
func runMetricsPrint(ctx context.Context, args []string) error {
	out, err := localClient.UserMetrics(ctx)
	if err != nil {
		return err
	}
	Stdout.Write(out)
	return nil
}

// runMetricsWrite writes metric values to a file.
func runMetricsWrite(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: lanhc metrics write <path>")
	}
	path := args[0]
	out, err := localClient.UserMetrics(ctx)
	if err != nil {
		return err
	}
	return atomicfile.WriteFile(path, out, 0644)
}
