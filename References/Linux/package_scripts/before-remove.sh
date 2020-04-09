#!/bin/sh

echo "[+] Disconnecting ..."
ivpn disconnect || echo "[-] Failed to disconnect"

echo "[+] Disabling firewall ..."
ivpn firewall -off || echo "[-] Failed to disable firewall"

echo "[+] Logging out ..."
ivpn logout || echo "[-] Failed to log out"

echo "[+] Service cleanup (pleaserun) ..."
sh /usr/share/pleaserun/ivpn-service/generate-cleanup.sh || echo "[-] Service cleanup FAILED!"