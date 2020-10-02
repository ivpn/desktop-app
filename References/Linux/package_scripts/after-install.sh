#!/bin/sh

echo "[*] After install (<%= version %> : <%= pkg %> : $1)"

DIR=/usr/share/applications
DESKTOP_FILE=$DIR/IVPN.desktop
if [ -d "$DIR" ]; then
    echo "[ ] Installing .desktop file..."
    ln -fs /opt/ivpn/ui/IVPN.desktop $DESKTOP_FILE || echo "[!] Failed to create .desktop file: '$DESKTOP_FILE'"
else
    echo "[!] Unable to install .desktop file. Folder '$DIR' not exists"
fi
