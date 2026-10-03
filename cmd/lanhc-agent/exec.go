// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"context"
	"fmt"
	"strings"
)

// Command templates. The agent never accepts a free-form shell string; callers
// reference a template by id and pass arguments. This keeps the audit surface
// small and makes every permitted action explicit.

type commandTemplate struct {
	ID          string
	Description string
	Args        []string
	NeedsParam  string // template argument name, e.g. "device"
}

var commandTemplates = map[string]commandTemplate{
	"smartctl-long": {
		ID:          "smartctl-long",
		Description: "Run an extended SMART self-test on a block device (read-only background test).",
		Args:        []string{"smartctl", "-t", "long", "%s"},
		NeedsParam:  "device",
	},
	"smartctl-info": {
		ID:          "smartctl-info",
		Description: "Print SMART info/health for a block device (read-only).",
		Args:        []string{"smartctl", "-H", "-i", "%s"},
		NeedsParam:  "device",
	},
}

// runTemplate executes a whitelisted command template. Every field is constrained:
// only a template id, a single parameter name and a short value are accepted.
func runTemplate(templateID, paramName, paramValue string) (string, error) {
	tpl, ok := commandTemplates[templateID]
	if !ok {
		return "", fmt.Errorf("unknown command template: %s", templateID)
	}
	if paramName != tpl.NeedsParam {
		return "", fmt.Errorf("template %s requires parameter %q", templateID, tpl.NeedsParam)
	}
	if len(paramValue) > 128 || paramValue == "" {
		return "", fmt.Errorf("parameter too long or empty")
	}
	if strings.ContainsAny(paramValue, " \t\n\r'\"`$;&|<>()") {
		return "", fmt.Errorf("parameter contains forbidden characters")
	}
	args := make([]string, len(tpl.Args))
	for i, a := range tpl.Args {
		if strings.Contains(a, "%s") {
			args[i] = fmt.Sprintf(a, paramValue)
		} else {
			args[i] = a
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*execTimeout)
	defer cancel()
	cmd := execCommandContext(ctx, args[0], args[1:]...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
