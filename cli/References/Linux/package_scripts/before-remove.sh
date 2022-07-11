#!/bin/sh

echo "[*] Before remove (<%= version %> : <%= pkg %> : $1)"

# When removing package: $1==0 for RPM; $1 == "remove" for DEB
_IS_REMOVE=0
if [ "$1" = "remove" -o "$1" = "0" ]; then    
  _IS_REMOVE=1
fi

echo "[+] Trying to disconnect (before-remove) ..."
/usr/bin/ivpn disconnect || echo "[-] Failed to disconnect"

if [ $_IS_REMOVE = 1 ]; then
    echo "[+] Disabling firewall (before-remove) ..."
    /usr/bin/ivpn firewall -off || echo "[-] Failed to disable firewall"

    echo "[+] Logging out (before-remove) ..."
    /usr/bin/ivpn logout || echo "[-] Failed to log out"

    echo "[+] Service cleanup (before-remove: pleaserun) ..."
    sh /usr/share/pleaserun/ivpn-service/generate-cleanup.sh || echo "[-] Service cleanup FAILED!"
fi
