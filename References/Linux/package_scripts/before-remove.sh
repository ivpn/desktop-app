#!/bin/sh

if [ -f /opt/ivpn/mutable/settings.json ]; then
    # In case of installing new version, we have to login back with current logged-in accountID after installation finished.
    # Therefore we are saving accountID into temporary file (will be deleted after 'after_remove' script execution)
    echo "[+] Preparing upgrade data ..."
    ACCID=$(cat /opt/ivpn/mutable/settings.json | grep -o \"AccountID\":\"[a-zA-Z0-9]*\" | cut -d '"' -f 4) || echo "[-] Failed to read accountID"
    if [ ! -z "$ACCID" ]; then
        echo $ACCID > /opt/ivpn/mutable/upgradeID.tmp || echo "[-] Failed to save accountID into temporary file"
    fi
fi

echo "[+] Disconnecting ..."
/usr/local/bin/ivpn disconnect || echo "[-] Failed to disconnect"

echo "[+] Disabling firewall ..."
/usr/local/bin/ivpn firewall -off || echo "[-] Failed to disable firewall"

echo "[+] Logging out ..."
/usr/local/bin/ivpn logout || echo "[-] Failed to log out"

echo "[+] Service cleanup (pleaserun) ..."
sh /usr/share/pleaserun/ivpn-service/generate-cleanup.sh || echo "[-] Service cleanup FAILED!"