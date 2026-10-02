// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

//go:build linux && !android && arm

package cli

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"os"
	"runtime"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"lanhc.com/version/distro"
)

func init() {
	maybeJetKVMConfigureCmd = jetKVMConfigureCmd
}

func jetKVMConfigureCmd() *ffcli.Command {
	if runtime.GOOS != "linux" || distro.Get() != distro.JetKVM {
		return nil
	}
	return &ffcli.Command{
		Name:       "jetkvm",
		Exec:       runConfigureJetKVM,
		ShortUsage: "lanhc configure jetkvm",
		ShortHelp:  "Configure JetKVM to run lanhcd at boot",
		LongHelp: strings.TrimSpace(`
This command configures the JetKVM host to run lanhcd at boot.
`),
		FlagSet: (func() *flag.FlagSet {
			fs := newFlagSet("jetkvm")
			return fs
		})(),
	}
}

func runConfigureJetKVM(ctx context.Context, args []string) error {
	if len(args) > 0 {
		return errors.New("unknown arguments")
	}
	if runtime.GOOS != "linux" || distro.Get() != distro.JetKVM {
		return errors.New("only implemented on JetKVM")
	}
	if err := os.MkdirAll("/userdata/init.d", 0755); err != nil {
		return errors.New("unable to create /userdata/init.d")
	}
	err := os.WriteFile("/userdata/init.d/S22lanhc", bytes.TrimLeft([]byte(`
#!/bin/sh
# /userdata/init.d/S22lanhc
# Start/stop lanhcd

case "$1" in
  start)
    /userdata/lanhc/lanhcd > /dev/null 2>&1 &
    ;;
  stop)
    killall lanhcd
    ;;
  *)
    echo "Usage: $0 {start|stop}"
    exit 1
    ;;
esac
`), "\n"), 0755)
	if err != nil {
		return err
	}

	if err := os.Symlink("/userdata/lanhc/lanhc", "/bin/lanhc"); err != nil {
		if !os.IsExist(err) {
			return err
		}
	}

	printf("Done. Now restart your JetKVM.\n")
	return nil
}
