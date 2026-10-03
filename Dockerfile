# Copyright (c) Tailscale Inc & contributors
# SPDX-License-Identifier: BSD-3-Clause

# Note that this Dockerfile is currently NOT used to build any of the published
# Lanhc container images and may have drifted from the image build mechanism
# we use.
# Lanhc images are currently built using https://github.com/tailscale/mkctr,
# and the build script can be found in ./build_docker.sh.
#
# If you want to build local images for testing, you can use make.
#
# To build a Lanhc image and push to the local docker registry:
#
#   $ REPO=local/lanhc TAGS=v0.0.1 PLATFORM=local  make publishdevimage
#
# To build a Lanhc image and push to a remote docker registry:
#
#   $ REPO=<your-registry>/<your-repo>/lanhc TAGS=v0.0.1  make publishdevimage
#
# This Dockerfile includes all the lanhc binaries.
#
# To build the Dockerfile:
#
#     $ docker build -t lanhc/lanhc .
#
# To run the lanhcd agent:
#
#     $ docker run -d --name=lanhcd -v /var/lib:/var/lib -v /dev/net/tun:/dev/net/tun --network=host --privileged lanhc/lanhc lanhcd
#
# To then log in:
#
#     $ docker exec lanhcd lanhc up
#
# To see status:
#
#     $ docker exec lanhcd lanhc status


FROM golang:1.26-alpine AS build-env

WORKDIR /go/src/lanhc

COPY go.mod go.sum ./
RUN go mod download

# Pre-build some stuff before the following COPY line invalidates the Docker cache.
RUN go install \
    github.com/aws/aws-sdk-go-v2/aws \
    github.com/aws/aws-sdk-go-v2/config \
    gvisor.dev/gvisor/pkg/tcpip/adapters/gonet \
    gvisor.dev/gvisor/pkg/tcpip/stack \
    golang.org/x/crypto/ssh \
    golang.org/x/crypto/acme \
    github.com/coder/websocket \
    github.com/mdlayher/netlink

COPY . .

# see build_docker.sh
ARG VERSION_LONG=""
ENV VERSION_LONG=$VERSION_LONG
ARG VERSION_SHORT=""
ENV VERSION_SHORT=$VERSION_SHORT
ARG VERSION_GIT_HASH=""
ENV VERSION_GIT_HASH=$VERSION_GIT_HASH
ARG TARGETARCH

RUN GOARCH=$TARGETARCH go install -ldflags="\
      -X lanhc.com/version.longStamp=$VERSION_LONG \
      -X lanhc.com/version.shortStamp=$VERSION_SHORT \
      -X lanhc.com/version.gitCommitStamp=$VERSION_GIT_HASH" \
      -v ./cmd/lanhc ./cmd/lanhcd ./cmd/containerboot

FROM alpine:3.22
RUN apk add --no-cache ca-certificates iptables iproute2 ip6tables
# Alpine 3.19 replaced legacy iptables with nftables based implementation.
# Lanhc is used on some hosts that don't support nftables, such as Synology
# NAS, so link iptables back to legacy version. Hosts that don't require legacy
# iptables should be able to use Lanhc in nftables mode.  See
# https://github.com/lanhc/lanhc/issues/17854
RUN rm /usr/sbin/iptables && ln -s /usr/sbin/iptables-legacy /usr/sbin/iptables
RUN rm /usr/sbin/ip6tables && ln -s /usr/sbin/ip6tables-legacy /usr/sbin/ip6tables

COPY --from=build-env /go/bin/* /usr/local/bin/
# For compat with the previous run.sh, although ideally you should be
# using build_docker.sh which sets an entrypoint for the image.
RUN mkdir /lanhc && ln -s /usr/local/bin/containerboot /lanhc/run.sh
