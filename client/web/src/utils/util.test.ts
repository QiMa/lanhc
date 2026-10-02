// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

import { isLanhcIPv6, pluralize } from "src/utils/util"
import { describe, expect, it } from "vitest"

describe("pluralize", () => {
  it("test routes", () => {
    expect(pluralize("route", "routes", 1)).toBe("route")
    expect(pluralize("route", "routes", 2)).toBe("routes")
  })
})

describe("isLanhcIPv6", () => {
  it("test ips", () => {
    expect(isLanhcIPv6("100.101.102.103")).toBeFalsy()
    expect(
      isLanhcIPv6("fd7a:115c:a1e0:ab11:1111:cd11:111e:f11g")
    ).toBeTruthy()
  })
})
