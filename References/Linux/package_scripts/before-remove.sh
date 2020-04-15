#!/bin/sh

echo "[+] Disconnecting ..."
/usr/local/bin/ivpn disconnect || echo "[-] Failed to disconnect"

echo "[+] Disabling firewall ..."
/usr/local/bin/ivpn firewall -off || echo "[-] Failed to disable firewall"

echo "[+] Logging out ..."
/usr/local/bin/ivpn logout || echo "[-] Failed to log out"

echo "[+] Service cleanup (pleaserun) ..."
sh /usr/share/pleaserun/ivpn-service/generate-cleanup.sh || echo "[-] Service cleanup FAILED!"