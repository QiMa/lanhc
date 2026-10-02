if [ "$1" = "configure" ] || [ "$1" = "abort-upgrade" ] || [ "$1" = "abort-deconfigure" ] || [ "$1" = "abort-remove" ] ; then
	  deb-systemd-helper unmask 'lanhc.nginx-auth.socket' >/dev/null || true
	  if deb-systemd-helper --quiet was-enabled 'lanhc.nginx-auth.socket'; then
		    deb-systemd-helper enable 'lanhc.nginx-auth.socket' >/dev/null || true
	  else
		    deb-systemd-helper update-state 'lanhc.nginx-auth.socket' >/dev/null || true
	  fi

    if systemctl is-active lanhc.nginx-auth.socket >/dev/null; then
        systemctl --system daemon-reload >/dev/null || true
        deb-systemd-invoke stop 'lanhc.nginx-auth.service' >/dev/null || true
        deb-systemd-invoke restart 'lanhc.nginx-auth.socket' >/dev/null || true
    fi
fi
