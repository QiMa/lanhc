#!/bin/bash

set -eu

# Clean up folders and files created during build.
function cleanup() {
	rm -rf /Lanhc/$ARCH
	rm -f /Lanhc/sed*
	rm -f /Lanhc/qpkg.cfg
}
trap cleanup EXIT

mkdir -p /Lanhc/$ARCH
cp /lanhcd /Lanhc/$ARCH/lanhcd
cp /lanhc /Lanhc/$ARCH/lanhc

sed "s/\$QPKG_VER/$TSTAG-$QNAPTAG/g" /Lanhc/qpkg.cfg.in >/Lanhc/qpkg.cfg

qbuild --root /Lanhc --build-arch $ARCH --build-dir /out
