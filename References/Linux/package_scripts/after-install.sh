#!/bin/sh

/usr/bin/echo "[*] After install (<%= version %> : <%= pkg %> : $1)"

DIR=/usr/share/applications
DESKTOP_FILE=$DIR/IVPN.desktop
if [ -d "$DIR" ]; then
    /usr/bin/echo "[ ] Installing .desktop file..."
    /usr/bin/ln -fs /opt/ivpn/ui/IVPN.desktop $DESKTOP_FILE || /usr/bin/echo "[!] Failed to create .desktop file: '$DESKTOP_FILE'"
else
    /usr/bin/echo "[!] Unable to install .desktop file. Folder '$DIR' not exists"
fi

/usr/bin/sudo /usr/bin/chmod 4755 /opt/ivpn/ui/bin/chrome-sandbox || /usr/bin/echo "[!] Failed to 'chmod' for '/opt/ivpn/ui/bin/chrome-sandbox'"
