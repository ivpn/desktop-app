#!/bin/sh

echo "[*] After remove (<%= version %> : <%= pkg %> : $1)"

# When removing package: $1==0 for RPM; $1 == "remove" for DEB
_IS_REMOVE=0
if [ "$1" = "remove" -o "$1" = "0" ]; then    
  _IS_REMOVE=1
fi

if [ $_IS_REMOVE = 0 ]; then
  echo "[ ] Upgrade detected. After-remove operations skipped"  
  exit 0
fi

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
if [ -d $IVPN_TMP ] ; then
  echo "[+] Removing other files ..."
  # Normally, all files which were installed, deleted automatically
  # But ivpn-service also writing to 'mutable' additional temporary files (uninstaller know nothing about them)
  # Therefore, we are completely removing all content of '/opt/ivpn/mutable'
  rm -rf $IVPN_TMP || echo "[-] Removing '$IVPN_TMP' folder failed"
  rm -rf $IVPN_LOG || echo "[-] Removing '$IVPN_LOG' folder failed"
  #remove 'ivpn' folder (if empy)
  silent sudo rmdir $IVPN_DIR
fi

try_systemd_stop
