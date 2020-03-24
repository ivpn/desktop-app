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

echo "[+] Removing other files ..."
# Normally, all files whic were installed aldo will be delete automatically
# But ivpn-service also writing to 'etc' additional temporary files (uninstaller know nothing about them)
# Therefore, ]we are completely removing all content or 'etc'
rm -rf /opt/ivpn || echo "[-] Removing files failed"

try_systemd_stop