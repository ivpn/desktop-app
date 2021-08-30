#!/bin/sh

echo "[*] Before install (<%= version %> : <%= pkg %> : $1)"

INSTALLED_BIN_FOLDER="/usr/bin/"
if [ ! -f ${INSTALLED_BIN_FOLDER}/ivpn ] && [ -f /usr/local/bin/ivpn ]; then
  # old installation path (used till v3.3.20)
  INSTALLED_BIN_FOLDER="/usr/local/bin/"
  echo "[ ] Detected old installation path: '$INSTALLED_BIN_FOLDER'"
fi

PKG_TYPE=<%= pkg %>
if [ "$PKG_TYPE" = "rpm" ]; then
  if [ -f ${INSTALLED_BIN_FOLDER}/ivpn ]; then
    # Skip running 'remove' scripts when upgrading
    mkdir -p /opt/ivpn/mutable
    echo "upgrade" > /opt/ivpn/mutable/rpm_upgrade.lock || echo "[-] Failed to save rpm_upgrade.lock"
  fi
fi

# DEB argument on upgrade - 'upgrade' (or 'configure' for after-install script); RPM - '1'
if [ "$1" = "upgrade" ] || [ "$1" = "1" ] ; then
  # let after-install script know that it is upgrade
  mkdir -p /opt/ivpn/mutable
  echo "upgrade" > /opt/ivpn/upgrade.lock || echo "[-] Failed to save upgrade.lock"
fi

if [ -f /opt/ivpn/mutable/upgradeID.tmp ]; then
    echo "[ ] Upgrade detected"
    mv /opt/ivpn/mutable/upgradeID.tmp /opt/ivpn/mutable/toUpgradeID.tmp || echo "[-] Failed to prepare accountID to re-login"
fi

if [ -f ${INSTALLED_BIN_FOLDER}/ivpn ]; then
  echo "[+] Trying to disable firewall (before install)..."
  ${INSTALLED_BIN_FOLDER}/ivpn firewall -off || echo "[-] Failed to disable firewall"

  echo "[+] Trying to disconnect (before install) ..."
  ${INSTALLED_BIN_FOLDER}/ivpn disconnect || echo "[-] Failed to disconnect"
fi
