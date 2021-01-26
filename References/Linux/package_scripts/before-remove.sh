#!/bin/sh

/usr/bin/echo "[*] Before remove (<%= pkg %>)"

PKG_TYPE=<%= pkg %>

/usr/bin/echo "[+] Disabling firewall (before remove) ..."
/usr/local/bin/ivpn firewall -off || /usr/bin/echo "[-] Failed to disable firewall"

/usr/bin/echo "[+] Disconnecting (before remove) ..."
/usr/local/bin/ivpn disconnect || /usr/bin/echo "[-] Failed to disconnect"

if [ "$PKG_TYPE" = "rpm" ]; then
    if [ -f /opt/ivpn/mutable/rpm_upgrade.lock ]; then
        /usr/bin/echo "[ ] Upgrade detected. Remove operations skipped"
        exit 0
    fi
fi

if [ -f /opt/ivpn/mutable/settings.json ]; then
    # In case of installing new version, we have to login back with current logged-in accountID after installation finished.
    # Therefore we are saving accountID into temporary file (will be deleted after 'after_install' script execution)
    /usr/bin/echo "[+] Preparing upgrade data ..."
    ACCID=$(/usr/bin/cat /opt/ivpn/mutable/settings.json | /usr/bin/grep -o \"AccountID\":\"[a-zA-Z0-9]*\" | /usr/bin/cut -d '"' -f 4) || /usr/bin/echo "[-] Failed to read accountID"
    if [ ! -z "$ACCID" ]; then
        /usr/bin/echo $ACCID > /opt/ivpn/mutable/upgradeID.tmp || /usr/bin/echo "[-] Failed to save accountID into temporary file"
    fi
fi

/usr/bin/echo "[+] Logging out ..."
/usr/local/bin/ivpn logout || /usr/bin/echo "[-] Failed to log out"

/usr/bin/echo "[+] Service cleanup (pleaserun) ..."
/usr/bin/sh /usr/share/pleaserun/ivpn-service/generate-cleanup.sh || /usr/bin/echo "[-] Service cleanup FAILED!"
