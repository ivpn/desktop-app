#!/bin/sh

echo "[+] Removing other files ..."
# Normally, all files whic were installed aldo will be delete automatically
# But ivpn-service also writing to 'etc' additional temporary files (uninstaller know nothing about them)
# Therefore, ]we are completely removing all content or 'etc'
rm -rf /opt/ivpn || echo "[-] Removing files failed"