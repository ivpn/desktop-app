#!/bin/sh

/usr/bin/echo "[*] After install (<%= pkg %>)"

NEED_TO_SAVE_INSTRUCTIONS=true
IVPN_ETC="/opt/ivpn/etc"
IVPN_TMP="/opt/ivpn/mutable"
INSTRUCTIONS_FILE="$IVPN_TMP/service_install.txt"
/usr/bin/mkdir -p $IVPN_TMP
[ -e $INSTRUCTIONS_FILE ] && /usr/bin/rm $INSTRUCTIONS_FILE


silent() {
  "$@" > /dev/null 2>&1
}

has_systemd() {
  # Some OS vendors put systemd in ... different places ...
  [ -d "/lib/systemd/system/" -o -d "/usr/lib/systemd/system" ] && silent /usr/bin/which systemctl
}

try_systemd_install() {
    if has_systemd ; then
        /usr/bin/echo "[ ] systemd detected. Trying to start service ..."
        /usr/bin/echo "[+] Stopping old service (if exists)"
        /usr/bin/systemctl stop ivpn-service
        /usr/bin/echo "[+] Enabling service"
        /usr/bin/systemctl enable ivpn-service || return 1
        /usr/bin/echo "[+] Starting service"
        /usr/bin/systemctl start ivpn-service || return 1

        NEED_TO_SAVE_INSTRUCTIONS=false
        return 0
    else
        /usr/bin/echo "[-] Unable to start service automatically"
    fi
}

/usr/bin/echo "[+] Defining access rights for files ..."
silent /usr/bin/chmod 0400 $IVPN_ETC/*             # can read only owner (root)
silent /usr/bin/chmod 0600 $IVPN_ETC/servers.json  # can read/wrire only owner (root)
silent /usr/bin/chmod 0700 $IVPN_ETC/*.sh          # can execute only owner (root)
silent /usr/bin/chmod 0700 $IVPN_ETC/*.up          # can execute only owner (root)
silent /usr/bin/chmod 0700 $IVPN_ETC/*.down        # can execute only owner (root)
silent /usr/bin/chmod 0755 /usr/local/bin/ivpn         # can change only owner (root)
silent /usr/bin/chmod 0755 /usr/local/bin/ivpn-service # can change only owner (root)


IVPN_SAVED_DNS_FILE="/etc/resolv.conf.ivpnsave"
if [ -f $IVPN_SAVED_DNS_FILE ]; then
  /usr/bin/echo "[+] restoring DNS configuration from previous installation"
  /usr/bin/mv $IVPN_SAVED_DNS_FILE /etc/resolv.conf || /usr/bin/echo "[-] Restoring DNS failed"
fi

/usr/bin/echo "[+] Service install start (pleaserun) ..."
INSTALL_OUTPUT=$(/usr/bin/sh /usr/share/pleaserun/ivpn-service/install.sh)
if [ $? -eq 0 ]; then
    # Print output of the install script
    /usr/bin/echo $INSTALL_OUTPUT

    try_systemd_install
else
    # Print output of the install script
    /usr/bin/echo $INSTALL_OUTPUT
    /usr/bin/echo "[-] Service install FAILED!"
fi

if $NEED_TO_SAVE_INSTRUCTIONS == true ; then
    /usr/bin/echo $INSTALL_OUTPUT > $INSTRUCTIONS_FILE
    /usr/bin/echo "[!] Service start instructions saved into file: '$INSTRUCTIONS_FILE'"
fi

FILE_ACCID_TO_UPGRADE="/opt/ivpn/mutable/toUpgradeID.tmp"
if [ -f $FILE_ACCID_TO_UPGRADE ]; then
  # It is an upgrade.
  # We need to re-login after installation finished.
  # Read account ID
  ACCID=$(/usr/bin/cat $FILE_ACCID_TO_UPGRADE) || /usr/bin/echo "[-] Finishing installation: Failed to read accountID to re-login"

  # do not forget to remove temporary file
  silent /usr/bin/rm $FILE_ACCID_TO_UPGRADE

  if [ ! -z "$ACCID" ]; then
    # giving a chance for a daemon to fully start
    /usr/bin/sleep 1
    /usr/bin/echo "[+] Logging in ..."
    /usr/local/bin/ivpn login $ACCID #||  echo "[-] Finishing installation: Failed to to re-login (try#1)"
    if [ ! $? -eq 0 ]; then
      /usr/bin/echo "[-] Finishing installation: Failed to to re-login (try#1)"
      /usr/bin/echo "[ ] Retry ..."
      /usr/bin/sleep 3
      /usr/local/bin/ivpn login $ACCID ||  /usr/bin/echo "[-] Finishing installation: Failed to to re-login (try#2)"
    fi
  fi
fi
