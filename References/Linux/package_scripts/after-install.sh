#!/bin/sh

NEED_TO_SAVE_INSTRUCTIONS=true
IVPN_ETC="/opt/ivpn/etc"
IVPN_TMP="/opt/ivpn/mutable"
INSTRUCTIONS_FILE="$IVPN_TMP/service_install.txt"
mkdir -p $IVPN_TMP
[ -e $INSTRUCTIONS_FILE ] && rm $INSTRUCTIONS_FILE


silent() {
  "$@" > /dev/null 2>&1
}

has_systemd() {
  # Some OS vendors put systemd in ... different places ...
  [ -d "/lib/systemd/system/" -o -d "/usr/lib/systemd/system" ] && silent which systemctl
}

try_systemd_install() {
    if has_systemd ; then
        echo "[ ] systemd detected. Trying to start service ..."
        echo "[+] Stopping old service (if exists)"
        systemctl stop ivpn-service
        echo "[+] Enabling service"
        systemctl enable ivpn-service || return 1
        echo "[+] Starting service"
        systemctl start ivpn-service || return 1

        NEED_TO_SAVE_INSTRUCTIONS=false
        return 0
    else
        echo "[-] Unable to start service automatically"
    fi
}

echo "[+] Defining access rights for files ..."
silent chmod 0400 $IVPN_ETC/*             # can read only owner (root)
silent chmod 0600 $IVPN_ETC/servers.json  # can read/wrire only owner (root)
silent chmod 0700 $IVPN_ETC/*.sh          # can execute only owner (root)
silent chmod 0700 $IVPN_ETC/*.up          # can execute only owner (root)
silent chmod 0700 $IVPN_ETC/*.down        # can execute only owner (root)


IVPN_SAVED_DNS_FILE="/etc/resolv.conf.ivpnsave"
if [ -f $IVPN_SAVED_DNS_FILE ]; then 
  echo "[+] restoring DNS configuration from previous installation"
  mv $IVPN_SAVED_DNS_FILE /etc/resolv.conf || echo "[-] Restoring DNS failed" 
fi

echo "[+] Service install start (pleaserun) ..."
INSTALL_OUTPUT=$(sh /usr/share/pleaserun/ivpn-service/install.sh) 
if [ $? -eq 0 ]; then 
    # Print output of the install script
    echo $INSTALL_OUTPUT

    try_systemd_install
else
    # Print output of the install script
    echo $INSTALL_OUTPUT
    echo "[-] Service install FAILED!"
fi

if $NEED_TO_SAVE_INSTRUCTIONS == true ; then
    echo $INSTALL_OUTPUT > $INSTRUCTIONS_FILE
    echo "[!] Service start instructions saved into file: '$INSTRUCTIONS_FILE'"
fi

FILE_ACCID_TO_UPGRADE="/opt/ivpn/mutable/toUpgradeID.tmp"
if [ -f $FILE_ACCID_TO_UPGRADE ]; then
  # It is an upgrade.
  # We need to re-login after installation finished.
  # Read account ID
  ACCID=$(cat $FILE_ACCID_TO_UPGRADE) || echo "[-] Finishing installation: Failed to read accountID to re-login"

  # do not forget to remove temporary file
  silent rm $FILE_ACCID_TO_UPGRADE

  if [ ! -z "$ACCID" ]; then
    # giving a chance for a daemon to fully start
    sleep 1
    echo "[+] Logging in ..."
    /usr/local/bin/ivpn login $ACCID #||  echo "[-] Finishing installation: Failed to to re-login (try#1)"
    if [ ! $? -eq 0 ]; then
      echo "[-] Finishing installation: Failed to to re-login (try#1)"
      echo "[ ] Retry ..."
      sleep 3
      /usr/local/bin/ivpn login $ACCID ||  echo "[-] Finishing installation: Failed to to re-login (try#2)"
    fi
  fi

fi