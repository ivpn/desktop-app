#!/bin/sh

/usr/bin/echo "[*] Before install (<%= pkg %>)"

PKG_TYPE=<%= pkg %>
if [ "$PKG_TYPE" = "rpm" ]; then
  if [ -f /usr/local/bin/ivpn ]; then
    # Necessary to check if the previous version can be upgraded.
    # Old versions have broken installer. It is not possible to upgrade correctly.
    BUILD_DATE=$(/usr/local/bin/ivpn -h | /usr/bin/grep -o date:[0-9]*-[0-9]*-[0-9]* | /usr/bin/cut -d ':' -f 2) || /usr/bin/echo "[-] Failed to determine build date of the old version"
    if [ $BUILD_DATE \< "2020-05-29" ]; then
        /usr/bin/echo "[!] Old version detected (date:$BUILD_DATE)"
        /usr/bin/echo "**************************************************"
        /usr/bin/echo "*    PLEASE, UNINSTALL THE OLD VERSION FIRST!    *"
        /usr/bin/echo "**************************************************"
        exit 1
    fi

    # Skip running 'remove' scripts when upgrading
    /usr/bin/mkdir -p /opt/ivpn/mutable
    /usr/bin/echo "upgrading" > /opt/ivpn/mutable/rpm_upgrade.lock || /usr/bin/echo "[-] Failed to save rpm_upgrade.lock"
  fi
fi

if [ -f /opt/ivpn/mutable/upgradeID.tmp ]; then
    /usr/bin/echo "[ ] Upgrade detected"
    /usr/bin/mv /opt/ivpn/mutable/upgradeID.tmp /opt/ivpn/mutable/toUpgradeID.tmp || /usr/bin/echo "[-] Failed to prepare accountID to re-login"
fi

if [ -f /usr/local/bin/ivpn ]; then
  /usr/bin/echo "[+] Trying to disable firewall (before install)..."
  /usr/local/bin/ivpn firewall -off || /usr/bin/echo "[-] Failed to disable firewall"

  /usr/bin/echo "[+] Trying to disconnect (before install) ..."
  /usr/local/bin/ivpn disconnect || /usr/bin/echo "[-] Failed to disconnect"
fi
