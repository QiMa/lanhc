// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

import React from "react"
import { useAPI } from "src/api"
import LanhcIcon from "src/assets/icons/lanhc-icon.svg?react"
import { NodeData } from "src/types"
import Button from "src/ui/button"

/**
 * LoginView is rendered when the client is not authenticated
 * to a tailnet.
 */
export default function LoginView({ data }: { data: NodeData }) {
  const api = useAPI()

  return (
    <div className="mb-8 py-6 px-8 bg-white rounded-md shadow-2xl">
      <LanhcIcon className="my-2 mb-8" />
      {data.Status === "Stopped" ? (
        <>
          <div className="mb-6">
            <h3 className="text-3xl font-semibold mb-3">Connect</h3>
            <p className="text-gray-700">
              你的设备已与蓝核AI智控台断开连接。
            </p>
          </div>
          <Button
            onClick={() => api({ action: "up", data: {} })}
            className="w-full mb-4"
            intent="primary"
          >
            连接到蓝核AI智控台
          </Button>
        </>
      ) : data.IPv4 ? (
        <>
          <div className="mb-6">
            <p className="text-gray-700">
              Your device’s key has expired. Reauthenticate this device by
              logging in again, or{" "}
              <a
                href="https://lanhc.com/kb/1028/key-expiry"
                className="link"
                target="_blank"
                rel="noreferrer"
              >
                learn more
              </a>
              .
            </p>
          </div>
          <Button
            onClick={() =>
              api({ action: "up", data: { Reauthenticate: true } })
            }
            className="w-full mb-4"
            intent="primary"
          >
            Reauthenticate
          </Button>
        </>
      ) : (
        <>
          <div className="mb-6">
            <h3 className="text-3xl font-semibold mb-3">Log in</h3>
            <p className="text-gray-700">
              登录你的蓝核AI智控台网络即可开始。
              Or,&nbsp;learn&nbsp;more at{" "}
              <a
                href="https://lanhc.com/"
                className="link"
                target="_blank"
                rel="noreferrer"
              >
                lanhc.com
              </a>
              .
            </p>
          </div>
          <Button
            onClick={() =>
              api({
                action: "up",
                data: {
                  Reauthenticate: true,
                },
              })
            }
            className="w-full mb-4"
            intent="primary"
          >
            Log In
          </Button>
        </>
      )}
    </div>
  )
}
