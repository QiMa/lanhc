#!/bin/sh
CONF=/etc/config/qpkg.conf
QPKG_NAME="Lanhc"
QPKG_ROOT=$(/sbin/getcfg ${QPKG_NAME} Install_Path -f ${CONF} -d"")
exec "${QPKG_ROOT}/lanhc" --socket=/tmp/lanhc/lanhcd.sock web --cgi --prefix="/cgi-bin/qpkg/Lanhc/index.cgi/"
