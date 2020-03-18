#!/bin/bash

echo "# Logging out ..."
ivpn logout || echo "# Logging out failed"

echo "# Service cleanup start (pleaserun) ..."
sh /usr/share/pleaserun/ivpn-service/generate-cleanup.sh || echo "# Service cleanup FAILED!"
echo "# ... Service cleanup end"

echo "# Removing logs ..."
rm -rf /opt/ivpn/log || echo "# Removing logs failed"

echo "# Removing other files ..."
# Normally, all files whic were installed aldo will be delete automatically
# But ivpn-service also writing to 'etc' additional temporary files (uninstaller know nothing about them)
# Therefore, we are completely removing all content or 'etc'
rm -rf /opt/ivpn || echo "# Removing files failed"

echo "#... Removing files end"