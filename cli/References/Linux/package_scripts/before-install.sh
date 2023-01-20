#!/bin/sh

echo "[*] Before install (<%= version %> : <%= pkg %> : $1)"

# Skip installation if 'ivpn' snap pachage already installed
snap list ivpn > /dev/null 2>&1 && echo "[!] INSTALLATION CANCELED: The snap package 'ivpn' is already installed. Please, uninstall the 'ivpn' snap package first." && exit 1

IVPN_BIN="/usr/bin/ivpn"
if [ ! -f ${IVPN_BIN} ] && [ -f /usr/local/bin/ivpn ]; then
  # old installation path (used till v3.3.20)
  IVPN_BIN="/usr/local/bin/ivpn"
  echo "[ ] Detected old installation path: '$IVPN_BIN'"
fi

if [ -f ${IVPN_BIN} ]; then
  #echo "[+] Trying to disable firewall (before install)..."
  #${IVPN_BIN} firewall -off || echo "[-] Failed to disable firewall"

  echo "[+] Trying to disconnect (before install) ..."
  ${IVPN_BIN} disconnect || echo "[-] Failed to disconnect"
fi

# ########################################################################################
#
# Next lines is in use only for compatibility with old package versions (v3.10.10 and older)
#
# ########################################################################################
# Folders changed:
# "/opt/ivpn/mutable" -> "/etc/opt/ivpn/mutable" 
# "/opt/ivpn/log"     -> "/var/log/ivpn" 
if [ -d /opt/ivpn/mutable ]; then 
  echo "[+] Migrating old-style mutable data from the previous installation ..."
  mkdir -p /etc/opt/ivpn
  mv /opt/ivpn/mutable /etc/opt/ivpn/mutable
fi
if [ -d /opt/ivpn/log ]; then 
  echo "[+] Migrating old-style logs from the previous installation ..." 
  mv /opt/ivpn/log /var/log/ivpn
fi

# ########################################################################################
#
# Next lines is in use only for compatibility with old package versions (v3.8.20 and older)
#
# ########################################################################################

# DEB: 'before-remove' script (old versions) saving account credentials into 'upgradeID.tmp' and doing logout,
# here we have to rename it to 'toUpgradeID.tmp' (to be compatible with old installation script style)
if [ -f /opt/ivpn/mutable/upgradeID.tmp ]; then
    echo "[ ] Upgrade detected (before-install: old-style)"
    mv /opt/ivpn/mutable/upgradeID.tmp /opt/ivpn/mutable/toUpgradeID.tmp || echo "[-] Failed to prepare accountID to re-login"
fi

# RPM: in order to sckip 'before-remove.sh \ after-remove.sh' scripts from the old-style installer
# we have to create file '/opt/ivpn/mutable/rpm_upgrade.lock'
if [ "<%= pkg %>" = "rpm" ]; then
  if [ -f ${IVPN_BIN} ]; then
    mkdir -p /opt/ivpn/mutable
    echo "upgrade" > /opt/ivpn/mutable/rpm_upgrade.lock || echo "[-] Failed to save rpm_upgrade.lock"
  fi
fi
