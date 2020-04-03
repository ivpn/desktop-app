#!/bin/sh

silent() {
  "$@" > /dev/null 2>&1
}

has_systemd() {
  # Some OS vendors put systemd in ... different places ...
  [ -d "/lib/systemd/system/" -o -d "/usr/lib/systemd/system" ] && silent which systemctl
}

try_systemd_stop() {
    if has_systemd ; then
        echo "[ ] systemd detected. Trying to stop service ..."
        
        echo "[+] Stopping service"
        silent systemctl stop ivpn-service

        echo "[+] Disabling service"
        silent systemctl disable ivpn-service   
        
        if [ -f "/etc/systemd/system/ivpn-service.service" ]; then
            echo "[+] Removing service"
            silent rm /etc/systemd/system/ivpn-service.service
        fi
        if [ -f "/usr/lib/systemd/system/ivpn-service.service" ]; then
            echo "[+] Removing service"
            silent rm /usr/lib/systemd/system/ivpn-service.service
        fi       
    fi
}

IVPN_DIR="/opt/ivpn"
IVPN_TMP="/opt/ivpn/mutable"
IVPN_LOG="/opt/ivpn/log"
IVPN_SAVED_DNS_FILE="/etc/resolv.conf.ivpnsave"
if [ -d $IVPN_TMP ] ; then
  echo "[+] Removing other files ..."
  # Normally, all files which were installed, deleted automatically
  # But ivpn-service also writing to 'etc' additional temporary files (uninstaller know nothing about them)
  # Therefore, we are completely removing all content of '/opt/ivpn/tmp'
  rm -rf $IVPN_TMP|| echo "[-] Removing '$IVPN_TMP' folder failed"
  rm -rf $IVPN_LOG|| echo "[-] Removing '$IVPN_LOG' folder failed"
  #remove 'ivpn' folder (if empyt)
  silent sudo rmdir $IVPN_DIR
fi 

if [ -f $IVPN_SAVED_DNS_FILE ]; then 
  echo "[+] restoring DNS configuration"
  mv $IVPN_SAVED_DNS_FILE /etc/resolv.conf || echo "[-] Restoring DNS failed" 
fi

try_systemd_stop