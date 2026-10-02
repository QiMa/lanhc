# $1 == 1 for initial installation.
# $1 == 2 for upgrades.

if [ $1 -eq 1 ] ; then
    # Normally, the lanhc-relay package would request shutdown of
    # its service before uninstallation. Unfortunately, the
    # lanhc-relay package we distributed doesn't have those
    # scriptlets. We definitely want relaynode to be stopped when
    # installing lanhcd though, so we blindly try to turn off
    # relaynode here.
    #
    # However, we also want this package installation to look like an
    # upgrade from relaynode! Therefore, if relaynode is currently
    # enabled, we want to also enable lanhcd. If relaynode is
    # currently running, we also want to start lanhcd.
    #
    # If there doesn't seem to be an active or enabled relaynode on
    # the system, we follow the RPM convention for package installs,
    # which is to not enable or start the service.
    relaynode_enabled=0
    relaynode_running=0
    if systemctl is-enabled lanhc-relay.service >/dev/null 2>&1; then
        relaynode_enabled=1
    fi
    if systemctl is-active lanhc-relay.service >/dev/null 2>&1; then
        relaynode_running=1
    fi

    systemctl --no-reload disable lanhc-relay.service >/dev/null 2>&1 || :
    systemctl stop lanhc-relay.service >/dev/null 2>&1 || :

    if [ $relaynode_enabled -eq 1 ]; then
        systemctl enable lanhcd.service >/dev/null 2>&1 || :
    else
        systemctl preset lanhcd.service >/dev/null 2>&1 || : 
    fi

    if [ $relaynode_running -eq 1 ]; then
        systemctl start lanhcd.service >/dev/null 2>&1 || :
    fi
fi 
