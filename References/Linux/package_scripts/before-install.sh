#!/bin/sh

if [ -f /opt/ivpn/mutable/upgradeID.tmp ]; then
    echo "[ ] Upgrade detected"
    mv /opt/ivpn/mutable/upgradeID.tmp /opt/ivpn/mutable/toUpgradeID.tmp || echo "[-] Failed to prepare accountID to re-login"
fi