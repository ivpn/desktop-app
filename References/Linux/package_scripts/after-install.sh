#!/bin/sh

NEED_TO_SAVE_INSTRUCTIONS=true
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

echo "[ ] Files installed"

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