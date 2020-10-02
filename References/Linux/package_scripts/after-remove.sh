#!/bin/sh

echo "[*] After remove (<%= version %> : <%= pkg %> : $1)"

# Obtaining information about user running the script
# (script can be executed with 'sudo', but we should get real user)
USER="${SUDO_USER:-$USER}"
UI_APP_USER_DIR="/home/${SUDO_USER:-$USER}/.config/ivpn-ui"

if [ -d $UI_APP_USER_DIR ] ; then
  if [ "$1" = "upgrade" ] ; then
    echo "[ ] Upgrade detected"
    echo "    Keeping application cache data from the previous version:"
    echo "    '$UI_APP_USER_DIR'"
  else
    echo "[ ] Removing application cache data from the previous version"
    rm -rf $UI_APP_USER_DIR || echo "[-] Failed to erase cache data in: '$UI_APP_USER_DIR'"
  fi
fi