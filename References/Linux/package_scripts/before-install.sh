#!/bin/sh

echo "[*] Before install (<%= pkg %>)"

PKG_TYPE=<%= pkg %>
if [ "$PKG_TYPE" = "rpm" ]; then
  if [ -f /usr/local/bin/ivpn ]; then
    # Necessary to check if the previous version can be upgraded.
    # Old versions have broken installer. It is not possible to upgrade correctly.
    BUILD_DATE=$(/usr/local/bin/ivpn -h | grep -o date:[0-9]*-[0-9]*-[0-9]* | cut -d ':' -f 2) || echo "[-] Failed to determine build date of the old version"
    if [ $BUILD_DATE \< "2020-05-29" ]; then
        echo "[!] Old version detected (date:$BUILD_DATE)"
        echo "**************************************************"
        echo "*    PLEASE, UNINSTALL THE OLD VERSION FIRST!    *"
        echo "**************************************************"
        exit 1
    fi

    # Skip running 'remove' scripts when upgrading
    mkdir -p /opt/ivpn/mutable
    echo "upgrading" > /opt/ivpn/mutable/rpm_upgrade.lock || echo "[-] Failed to save rpm_upgrade.lock"
  fi 
fi

if [ -f /opt/ivpn/mutable/upgradeID.tmp ]; then
    echo "[ ] Upgrade detected"
    mv /opt/ivpn/mutable/upgradeID.tmp /opt/ivpn/mutable/toUpgradeID.tmp || echo "[-] Failed to prepare accountID to re-login"
fi