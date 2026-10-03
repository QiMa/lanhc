// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

import React from "react"
import LanhcIcon from "src/assets/icons/lanhc-icon.svg?react"

/**
 * DisconnectedView is rendered after node logout.
 */
export default function DisconnectedView() {
  return (
    <>
      <LanhcIcon className="mx-auto" />
      <p className="mt-12 text-center text-text-muted">
        你已从该设备退出登录。要重新连接，需要通过蓝核AI智控台应用或
        蓝核AI智控台命令行界面重新认证设备。
      </p>
    </>
  )
}
