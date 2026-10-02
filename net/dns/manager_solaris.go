// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package dns

import (
	"lanhc.com/control/controlknobs"
	"lanhc.com/health"
	"lanhc.com/types/logger"
	"lanhc.com/util/eventbus"
	"lanhc.com/util/syspolicy/policyclient"
)

func NewOSConfigurator(logf logger.Logf, health *health.Tracker, bus *eventbus.Bus, _ policyclient.Client, _ *controlknobs.Knobs, iface string) (OSConfigurator, error) {
	return newDirectManager(logf, health, bus), nil
}
