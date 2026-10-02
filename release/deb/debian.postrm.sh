#!/bin/sh
set -e
if [ -d /run/systemd/system ] ; then
	systemctl --system daemon-reload >/dev/null || true
fi

if [ -x "/usr/bin/deb-systemd-helper" ]; then
    if [ "$1" = "remove" ]; then
		deb-systemd-helper mask 'lanhcd.service' >/dev/null || true
	fi

    if [ "$1" = "purge" ]; then
		deb-systemd-helper purge 'lanhcd.service' >/dev/null || true
		deb-systemd-helper unmask 'lanhcd.service' >/dev/null || true
		rm -rf /var/lib/lanhc
	fi
fi
