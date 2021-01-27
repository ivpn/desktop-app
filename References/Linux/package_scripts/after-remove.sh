#!/bin/sh

/usr/bin/echo "[*] After remove (<%= version %> : <%= pkg %> : $1)"

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
/usr/bin/echo "[+] Checking for 'ivpn-ui' running processes ..."
/usr/bin/ps aux | /usr/bin/grep /opt/ivpn/ui/bin/ivpn-ui | /usr/bin/grep -v grep  > /dev/null 2>&1
if [ $? -eq 0 ]; then
  /usr/bin/echo "[!] Detected: IVPN app is running"

  # We should be careful here: WE SHOULD NOT KILL THIS SCRIPT :)
  # (which also can have 'ivpn-ui' in process description)
  silent /usr/bin/kill -TERM $(/usr/bin/ps aux | /usr/bin/grep /opt/ivpn/ui/bin/ivpn-ui | /usr/bin/grep -v grep | /usr/bin/awk '{print $2}')
  silent /usr/bin/sleep 2
  silent /usr/bin/kill -KILL $(/usr/bin/ps aux | /usr/bin/grep /opt/ivpn/ui/bin/ivpn-ui | /usr/bin/grep -v grep | /usr/bin/awk '{print $2}')
fi

# DEB argument on upgrade - 'upgrade'; RPM - '1'
if [ "$1" = "upgrade" ] || [ "$1" = "1" ] ; then
  # UPGRADE
  if [ -d $UI_APP_USER_DIR ] ; then
    /usr/bin/echo "[!] Upgrade detected"
    /usr/bin/echo "    Keeping application cache data from the previous version:"
    /usr/bin/echo "    '$UI_APP_USER_DIR'"
  fi
else
  # REMOVE
  if [ -f $DESKTOP_FILE ]; then
    /usr/bin/echo "[+] Uninstalling .desktop file: '$DESKTOP_FILE' ..."
    /usr/bin/rm $DESKTOP_FILE || /usr/bin/echo "[-] Failed"
  fi

  if [ -d $UI_APP_USER_DIR ] ; then
    /usr/bin/echo "[+] Removing application cache data: '$UI_APP_USER_DIR' ..."
    /usr/bin/rm -rf $UI_APP_USER_DIR || /usr/bin/echo "[-] Failed"
  fi

  if [ -f $AUTOSTART_FILE ]; then
    /usr/bin/echo "[+] Removing application autostart file: '$AUTOSTART_FILE' ..."
    /usr/bin/rm $AUTOSTART_FILE || /usr/bin/echo "[-] Failed"
  fi

fi
