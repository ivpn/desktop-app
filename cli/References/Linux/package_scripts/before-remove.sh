#!/bin/sh

echo "[*] Before remove (<%= version %> : <%= pkg %> : $1)"

PKG_TYPE=<%= pkg %>

echo "[+] Disabling firewall (before remove) ..."
/usr/local/bin/ivpn firewall -off || echo "[-] Failed to disable firewall"

echo "[+] Disconnecting (before remove) ..."
/usr/local/bin/ivpn disconnect || echo "[-] Failed to disconnect"

if [ "$PKG_TYPE" = "rpm" ]; then
    if [ -f /opt/ivpn/mutable/rpm_upgrade.lock ]; then
        echo "[ ] Upgrade detected. Remove operations skipped"
        exit 0
    fi
fi

if [ -f /opt/ivpn/mutable/settings.json ]; then
    # In case of installing new version, we have to login back with current logged-in accountID after installation finished.
    # Therefore we are saving accountID into temporary file (will be deleted after 'after_install' script execution)
    echo "[+] Preparing upgrade data ..."
    ACCID=$(cat /opt/ivpn/mutable/settings.json | grep -o \"AccountID\":\"[a-zA-Z0-9]*\" | cut -d '"' -f 4) || echo "[-] Failed to read accountID"
    if [ ! -z "$ACCID" ]; then
        echo $ACCID > /opt/ivpn/mutable/upgradeID.tmp || echo "[-] Failed to save accountID into temporary file"
    fi
fi

echo "[+] Logging out ..."
/usr/local/bin/ivpn logout || echo "[-] Failed to log out"

echo "[+] Service cleanup (pleaserun) ..."
sh /usr/share/pleaserun/ivpn-service/generate-cleanup.sh || echo "[-] Service cleanup FAILED!"
