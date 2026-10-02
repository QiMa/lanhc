// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package jsonoutput_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"lanhc.com/cmd/lanhc/cli/jsonoutput"
)

func ExampleDNSStatusResult() {
	cmd := exec.Command("lanhc", "dns", "status", "--json")
	out, err := cmd.Output()
	if err != nil {
		if err, ok := errors.AsType[*exec.ExitError](err); ok {
			fmt.Fprintf(os.Stderr, "%s", err.Stderr)
		}
		panic(err)
	}

	var dnsStatus jsonoutput.DNSStatusResult
	if err := json.Unmarshal(out, &dnsStatus); err != nil {
		panic(err)
	}
	fmt.Printf("{accept-dns: %t, resolvers: %q}\n", dnsStatus.LanhcDNS, dnsStatus.Resolvers)
}
