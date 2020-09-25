#!/bin/sh

echo "[*] After install (<%= pkg %>)"
echo "[ ] Installing .desktop file..."
DESKTOP_FILE="/usr/share/applications/IVPN.desktop"
ln -fs /opt/ivpn/ui/IVPN.desktop $DESKTOP_FILE