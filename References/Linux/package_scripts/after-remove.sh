#!/bin/sh

echo "[*] After remove (<%= version %> : <%= pkg %> : $1)"

DESKTOP_FILE=/usr/share/applications/IVPN.desktop

# Obtaining information about user running the script
# (script can be executed with 'sudo', but we should get real user)
USER="${SUDO_USER:-$USER}"
UI_APP_USER_DIR="/home/${SUDO_USER:-$USER}/.config/ivpn-ui"

# DEB argument on upgrade - 'upgrade'; RPM - '1'
if [ "$1" = "upgrade" ] || [ "$1" = "1" ] ; then
  # UPGRADE
  if [ -d $UI_APP_USER_DIR ] ; then
    echo "[ ] Upgrade detected"
    echo "    Keeping application cache data from the previous version:"
    echo "    '$UI_APP_USER_DIR'"
  fi
else
  # REMOVE
  if [ -f $DESKTOP_FILE ]; then
    echo "[ ] Uninstalling .desktop file"
    rm $DESKTOP_FILE || echo "[-] Failed to remove file: '$DESKTOP_FILE'"
  fi

  if [ -d $UI_APP_USER_DIR ] ; then
    echo "[ ] Removing application cache data from the previous version"
    rm -rf $UI_APP_USER_DIR || echo "[-] Failed to erase cache data in: '$UI_APP_USER_DIR'"
  fi
fi

