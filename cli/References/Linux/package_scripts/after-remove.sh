#!/bin/sh

echo "[*] After remove (<%= version %> : <%= pkg %> : $1)"

PKG_TYPE=<%= pkg %>
if [ "$PKG_TYPE" = "rpm" ]; then
    if [ -f /opt/ivpn/mutable/rpm_upgrade.lock ]; then
        echo "[ ] Upgrade detected. Remove operations skipped"
        rm /opt/ivpn/mutable/rpm_upgrade.lock || echo "[-] Failed to remove rpm_upgrade.lock"
        exit 0
    fi
fi

silent() {
  "$@" > /dev/null 2>&1
}

has_systemd() {
  # Some OS vendors put systemd in ... different places ...
  [ -d "/lib/systemd/system/" -o -d "/usr/lib/systemd/system" ] && silent which systemctl
}

try_systemd_stop() {
    if has_systemd ; then
        echo "[ ] systemd detected. Trying to stop service ..."

        echo "[+] Stopping service"
        silent systemctl stop ivpn-service

        echo "[+] Disabling service"
        silent systemctl disable ivpn-service

        if [ -f "/etc/systemd/system/ivpn-service.service" ]; then
            echo "[+] Removing service"
            silent rm /etc/systemd/system/ivpn-service.service
        fi
        if [ -f "/usr/lib/systemd/system/ivpn-service.service" ]; then
            echo "[+] Removing service"
            silent rm /usr/lib/systemd/system/ivpn-service.service
        fi
    fi
}

FILE_ACCID_TO_UPGRADE="/opt/ivpn/mutable/toUpgradeID.tmp"
if [ -f $FILE_ACCID_TO_UPGRADE ]; then
  # It is an upgrade.
  # We need to re-login after installation finished.
  # Therefore we should not remove info about account ID.
  # Read into temporary variable
  ACCID=$(cat $FILE_ACCID_TO_UPGRADE) || echo "[-] Failed to read accountID to re-login"
fi

IVPN_DIR="/opt/ivpn"
IVPN_TMP="/opt/ivpn/mutable"
IVPN_LOG="/opt/ivpn/log"
IVPN_ETC="/opt/ivpn/etc"
if [ -d $IVPN_TMP ] ; then
  echo "[+] Removing other files ..."
  # Normally, all files which were installed, deleted automatically
  # But ivpn-service also writing to 'mutable' additional temporary files (uninstaller know nothing about them)
  # Therefore, we are completely removing all content of '/opt/ivpn/mutable'
  rm -rf $IVPN_TMP|| echo "[-] Removing '$IVPN_TMP' folder failed"
  rm -rf $IVPN_LOG|| echo "[-] Removing '$IVPN_LOG' folder failed"
  #rm -rf $IVPN_ETC|| echo "[-] Removing '$IVPN_ETC' folder failed"
  #rm -rf $IVPN_DIR|| echo "[-] Removing '$IVPN_DIR' folder failed"
  #remove 'ivpn' folder (if empy)
  silent sudo rmdir $IVPN_DIR
fi

if [ ! -z "$ACCID" ]; then
  # It is an upgrade.
  # We need to re-login after installation finished.
  # Therefore we should not remove info about account ID
  # Save to a file from temporary variable
    DIR=$(dirname $FILE_ACCID_TO_UPGRADE) || echo "[-] Failed to save accountID to re-login (1)"
    mkdir -p $DIR                         || echo "[-] Failed to save accountID to re-login (2)"
    echo $ACCID > $FILE_ACCID_TO_UPGRADE  || echo "[-] Failed to save accountID to re-login (3)"
  fi

IVPN_SAVED_DNS_FILE="/etc/resolv.conf.ivpnsave"
if [ -f $IVPN_SAVED_DNS_FILE ]; then
  echo "[+] restoring DNS configuration"
  mv $IVPN_SAVED_DNS_FILE /etc/resolv.conf || echo "[-] Restoring DNS failed"
fi

try_systemd_stop
