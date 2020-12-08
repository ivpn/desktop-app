#!/bin/sh

echo "[*] After remove (<%= version %> : <%= pkg %> : $1)"

# Obtaining information about user running the script
# (script can be executed with 'sudo', but we should get real user)
USER="${SUDO_USER:-$USER}"
UI_APP_USER_DIR="/home/${USER}/.config/ivpn-ui"
AUTOSTART_FILE="/home/${USER}/.config/autostart/ivpn-ui.desktop"
DESKTOP_FILE=/usr/share/applications/IVPN.desktop

silent() {
  "$@" > /dev/null 2>&1
}

# STOPPING APPLICATION (same functionality implemented also in 'before-install.sh')
echo "[+] Checking for 'ivpn-ui' running processes ..."
ps aux | grep /opt/ivpn/ui/bin/ivpn-ui | grep -v grep  > /dev/null 2>&1
if [ $? -eq 0 ]; then
  echo "[!] Detected: IVPN app is running"
  echo "[+] Disconnecting (if connected) ..."
  /usr/local/bin/ivpn disconnect || echo "[-] Failed to disconnect"
  echo "[+] Disabling firewall (if enabled) ..."
  /usr/local/bin/ivpn firewall -off || echo "[-] Failed to disable firewall"
  echo "[+] Killing all 'ivpn-ui' processes ..."
  # We should be careful here: WE SHOULD NOT KILL THIS SCRIPT :)
  # (which also can have 'ivpn-ui' in process description)
  silent kill -TERM $(ps aux | grep /opt/ivpn/ui/bin/ivpn-ui | grep -v grep | awk '{print $2}')
  silent sleep 2
  silent kill -KILL $(ps aux | grep /opt/ivpn/ui/bin/ivpn-ui | grep -v grep | awk '{print $2}')
fi

# DEB argument on upgrade - 'upgrade'; RPM - '1'
if [ "$1" = "upgrade" ] || [ "$1" = "1" ] ; then
  # UPGRADE
  if [ -d $UI_APP_USER_DIR ] ; then
    echo "[!] Upgrade detected"
    echo "    Keeping application cache data from the previous version:"
    echo "    '$UI_APP_USER_DIR'"
  fi
else
  # REMOVE
  if [ -f $DESKTOP_FILE ]; then
    echo "[+] Uninstalling .desktop file: '$DESKTOP_FILE' ..."
    rm $DESKTOP_FILE || echo "[-] Failed"
  fi

  if [ -d $UI_APP_USER_DIR ] ; then
    echo "[+] Removing application cache data: '$UI_APP_USER_DIR' ..."
    rm -rf $UI_APP_USER_DIR || echo "[-] Failed"
  fi

  if [ -f $AUTOSTART_FILE ]; then
    echo "[+] Removing application autostart file: '$AUTOSTART_FILE' ..."
    rm $AUTOSTART_FILE || echo "[-] Failed"
  fi

fi
