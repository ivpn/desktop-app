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
        systemctl stop ivpn-service

        echo "[+] Disabling service"
        systemctl disable ivpn-service   

        if [ -d "/lib/systemd/system/" ] ; then
            echo "[+] Removing service"
            silent rm /etc/systemd/system/ivpn-service.service
            silent rm /etc/systemd/system/ivpn-service.service
        fi 

        if [ -d "/usr/lib/systemd/system" ] ; then
            echo "[+] Removing service"
            silent rm /usr/lib/systemd/system/ivpn-service.service
            silent rm /usr/lib/systemd/system/ivpn-service.service
        fi     
    fi
}

IVPN_DIR="/opt/ivpn"
IVPN_TMP="/opt/ivpn/mutable"
IVPN_LOG="/opt/ivpn/log"
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

try_systemd_stop