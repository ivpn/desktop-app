#!/bin/sh

/usr/bin/echo "[*] After remove (<%= pkg %>)"

PKG_TYPE=<%= pkg %>
if [ "$PKG_TYPE" = "rpm" ]; then
    if [ -f /opt/ivpn/mutable/rpm_upgrade.lock ]; then
        /usr/bin/echo "[ ] Upgrade detected. Remove operations skipped"
        /usr/bin/rm /opt/ivpn/mutable/rpm_upgrade.lock || /usr/bin/echo "[-] Failed to remove rpm_upgrade.lock"
        exit 0
    fi
fi

silent() {
  "$@" > /dev/null 2>&1
}

has_systemd() {
  # Some OS vendors put systemd in ... different places ...
  [ -d "/lib/systemd/system/" -o -d "/usr/lib/systemd/system" ] && silent /usr/bin/which systemctl
}

try_systemd_stop() {
    if has_systemd ; then
        /usr/bin/echo "[ ] systemd detected. Trying to stop service ..."

        /usr/bin/echo "[+] Stopping service"
        silent /usr/bin/systemctl stop ivpn-service

        /usr/bin/echo "[+] Disabling service"
        silent /usr/bin/systemctl disable ivpn-service

        if [ -f "/etc/systemd/system/ivpn-service.service" ]; then
            /usr/bin/echo "[+] Removing service"
            silent /usr/bin/rm /etc/systemd/system/ivpn-service.service
        fi
        if [ -f "/usr/lib/systemd/system/ivpn-service.service" ]; then
            /usr/bin/echo "[+] Removing service"
            silent /usr/bin/rm /usr/lib/systemd/system/ivpn-service.service
        fi
    fi
}

FILE_ACCID_TO_UPGRADE="/opt/ivpn/mutable/toUpgradeID.tmp"
if [ -f $FILE_ACCID_TO_UPGRADE ]; then
  # It is an upgrade.
  # We need to re-login after installation finished.
  # Therefore we should not remove info about account ID.
  # Read into temporary variable
  ACCID=$(/usr/bin/cat $FILE_ACCID_TO_UPGRADE) || /usr/bin/echo "[-] Failed to read accountID to re-login"
fi

IVPN_DIR="/opt/ivpn"
IVPN_TMP="/opt/ivpn/mutable"
IVPN_LOG="/opt/ivpn/log"
IVPN_ETC="/opt/ivpn/etc"
if [ -d $IVPN_TMP ] ; then
  /usr/bin/echo "[+] Removing other files ..."
  # Normally, all files which were installed, deleted automatically
  # But ivpn-service also writing to 'mutable' additional temporary files (uninstaller know nothing about them)
  # Therefore, we are completely removing all content of '/opt/ivpn/mutable'
  /usr/bin/rm -rf $IVPN_TMP|| /usr/bin/echo "[-] Removing '$IVPN_TMP' folder failed"
  /usr/bin/rm -rf $IVPN_LOG|| /usr/bin/echo "[-] Removing '$IVPN_LOG' folder failed"
  #rm -rf $IVPN_ETC|| echo "[-] Removing '$IVPN_ETC' folder failed"
  #rm -rf $IVPN_DIR|| echo "[-] Removing '$IVPN_DIR' folder failed"
  #remove 'ivpn' folder (if empy)
  silent /usr/bin/sudo /usr/bin/rmdir $IVPN_DIR
fi

if [ ! -z "$ACCID" ]; then
  # It is an upgrade.
  # We need to re-login after installation finished.
  # Therefore we should not remove info about account ID
  # Save to a file from temporary variable
    DIR=$(/usr/bin/dirname $FILE_ACCID_TO_UPGRADE) || /usr/bin/echo "[-] Failed to save accountID to re-login (1)"
    /usr/bin/mkdir -p $DIR                         || /usr/bin/echo "[-] Failed to save accountID to re-login (2)"
    /usr/bin/echo $ACCID > $FILE_ACCID_TO_UPGRADE  || /usr/bin/echo "[-] Failed to save accountID to re-login (3)"
  fi

IVPN_SAVED_DNS_FILE="/etc/resolv.conf.ivpnsave"
if [ -f $IVPN_SAVED_DNS_FILE ]; then
  /usr/bin/echo "[+] restoring DNS configuration"
  /usr/bin/mv $IVPN_SAVED_DNS_FILE /etc/resolv.conf || /usr/bin/echo "[-] Restoring DNS failed" 
fi

try_systemd_stop
