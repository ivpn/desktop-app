#!/bin/bash

echo "# Service install start (pleaserun) ..."
sh /usr/share/pleaserun/ivpn-service/install.sh || echo "# Service install FAILED!"
echo "# ... Service install end"