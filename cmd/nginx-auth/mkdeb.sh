#!/usr/bin/env bash

set -e

VERSION=0.1.3
for ARCH in amd64 arm64; do
    CGO_ENABLED=0 GOARCH=${ARCH} GOOS=linux go build -o lanhc.nginx-auth .

    mkpkg \
        --out=lanhc-nginx-auth-${VERSION}-${ARCH}.deb \
        --name=lanhc-nginx-auth \
        --version=${VERSION} \
        --type=deb \
        --arch=${ARCH} \
        --postinst=deb/postinst.sh \
        --postrm=deb/postrm.sh \
        --prerm=deb/prerm.sh \
        --description="Lanhc NGINX authentication protocol handler" \
        --files=./lanhc.nginx-auth:/usr/sbin/lanhc.nginx-auth,./lanhc.nginx-auth.socket:/lib/systemd/system/lanhc.nginx-auth.socket,./lanhc.nginx-auth.service:/lib/systemd/system/lanhc.nginx-auth.service,./README.md:/usr/share/lanhc/nginx-auth/README.md

    mkpkg \
        --out=lanhc-nginx-auth-${VERSION}-${ARCH}.rpm \
        --name=lanhc-nginx-auth \
        --version=${VERSION} \
        --type=rpm \
        --arch=${ARCH} \
        --postinst=rpm/postinst.sh \
        --postrm=rpm/postrm.sh \
        --prerm=rpm/prerm.sh \
        --description="Lanhc NGINX authentication protocol handler" \
        --files=./lanhc.nginx-auth:/usr/sbin/lanhc.nginx-auth,./lanhc.nginx-auth.socket:/lib/systemd/system/lanhc.nginx-auth.socket,./lanhc.nginx-auth.service:/lib/systemd/system/lanhc.nginx-auth.service,./README.md:/usr/share/lanhc/nginx-auth/README.md
done
