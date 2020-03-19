#!/bin/sh

echo "[+] Logging out ..."
ivpn logout || echo "[-] Logging out failed"

echo "[+] Service cleanup start (pleaserun) ..."
sh /usr/share/pleaserun/ivpn-service/generate-cleanup.sh || echo "[-] Service cleanup FAILED!"