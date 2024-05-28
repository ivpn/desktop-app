#!/bin/sh

echo "[*] After install (<%= version %> : <%= pkg %> : $1)"

# update permissions for .desktop file and icon
DESKTOP_SRC_FILE=/opt/ivpn/ui/IVPN.desktop
ICON_SRC_FILE=/opt/ivpn/ui/ivpnicon.svg
sudo chmod 744 $DESKTOP_SRC_FILE
sudo chmod 744 $ICON_SRC_FILE

# create link to .desktop file
APPS_DIR=/usr/share/applications
DESKTOP_APPS_FILE=$APPS_DIR/IVPN.desktop
if [ -d "$APPS_DIR" ]; then
    echo "[ ] Installing .desktop file..."
    ln -fs $DESKTOP_SRC_FILE $DESKTOP_APPS_FILE || echo "[!] Failed to create link to .desktop file: '$DESKTOP_SRC_FILE' -> '$DESKTOP_APPS_FILE'"
else
    echo "[!] Unable to install .desktop file. Folder '$APPS_DIR' not exists"
fi

# electron app requires 'chrome-sandbox' to be setuid root in some systems
sudo chmod 4755 /opt/ivpn/ui/bin/chrome-sandbox || echo "[!] Failed to 'chmod' for '/opt/ivpn/ui/bin/chrome-sandbox'"