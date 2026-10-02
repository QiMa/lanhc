// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package cli

import (
	"context"
	"errors"

	"github.com/peterbourgon/ff/v3/ffcli"
	"lanhc.com/envknob"
)

var idTokenCmd = &ffcli.Command{
	Name:       "id-token",
	ShortUsage: "lanhc id-token <aud>",
	ShortHelp:  "Fetch an OIDC id-token for the Lanhc machine",
	LongHelp:   hidden,
	Exec:       runIDToken,
}

func runIDToken(ctx context.Context, args []string) error {
	if !envknob.UseWIPCode() {
		return errors.New("lanhc id-token: works-in-progress require LANHC_USE_WIP_CODE=1 envvar")
	}
	if len(args) != 1 {
		return errors.New("usage: lanhc id-token <aud>")
	}

	tr, err := localClient.IDToken(ctx, args[0])
	if err != nil {
		return err
	}

	outln(tr.IDToken)
	return nil
}
