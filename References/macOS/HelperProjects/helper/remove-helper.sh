#!/bin/sh
sudo launchctl unload /Library/LaunchDaemons/net.ivpn.client.Helper.plist
sudo rm /Library/LaunchDaemons/net.ivpn.client.Helper.plist
sudo rm /Library/PrivilegedHelperTools/net.ivpn.client.Helper
