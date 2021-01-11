#!/bin/bash

#save current dir
_BASE_DIR="$( pwd )"
_SCRIPT=`basename "$0"`
#enter the script folder
cd "$(dirname "$0")"
_SCRIPT_DIR="$( pwd )"

# check result of last executed command
function CheckLastResult
{
  if ! [ $? -eq 0 ]; then #check result of last command
    if [ -n "$1" ]; then
      echo $1
    else
      echo "FAILED"
    fi
    exit 1
  fi
}

# The Apple DevID certificate which will be used to sign IVPN Agent (Daemon) binary
# The helper will check IVPN Agent signature with this value
_SIGN_CERT="" #"WQXXM75BYN" # default value. Specific value can be passed by command-line argument: -c <APPLE_DEVID_SERT>
while getopts ":c:" opt; do
  case $opt in
    c) _SIGN_CERT="$OPTARG"
    ;;
  esac
done

if [ -z "${_SIGN_CERT}" ]; then
  echo "Usage:"
  echo "    $0 -c <APPLE_DEVID_CERT>"
  exit 1
fi

if [ ! -f "../helper/net.ivpn.client.Helper" ]; then
  echo " File not exists '../helper/net.ivpn.client.Helper'. Please, compile helper project first."
  exit 1
fi

rm -fr bin
CheckLastResult

echo "[ ] *** Compiling IVPN Installer / Uninstaller ***"

echo "[+] IVPN Installer: updating certificate info in .plist ..."
echo "    Apple DevID certificate: '${_SIGN_CERT}'"
plutil -replace SMPrivilegedExecutables -xml \
        "<dict> \
      		<key>net.ivpn.client.Helper</key> \
      		<string>identifier net.ivpn.client.Helper and certificate leaf[subject.OU] = ${_SIGN_CERT}</string> \
      	</dict>" "IVPN Installer-Info.plist" || CheckLastResult
plutil -replace SMPrivilegedExecutables -xml \
        "<dict> \
          <key>net.ivpn.client.Helper</key> \
          <string>identifier net.ivpn.client.Helper and certificate leaf[subject.OU] = ${_SIGN_CERT}</string> \
        </dict>" "IVPN Uninstaller-Info.plist" || CheckLastResult

echo "[+] IVPN Installer: make ..."
make
CheckLastResult

echo "[+] IVPN Installer: IVPN Installer.app ..."
mkdir -p "bin/IVPN Installer.app/Contents/Library/LaunchServices" || CheckLastResult
mkdir -p "bin/IVPN Installer.app/Contents/MacOS" || CheckLastResult
cp "../helper/net.ivpn.client.Helper" "bin/IVPN Installer.app/Contents/Library/LaunchServices" || CheckLastResult
cp "bin/IVPN Installer" "bin/IVPN Installer.app/Contents/MacOS" || CheckLastResult
cp "IVPN Installer-Info.plist" "bin/IVPN Installer.app/Contents/Info.plist" || CheckLastResult

echo "[+] IVPN Installer: IVPN Uninstaller.app ..."
mkdir -p "bin/IVPN Uninstaller.app/Contents/MacOS" || CheckLastResult
cp "bin/IVPN Uninstaller" "bin/IVPN Uninstaller.app/Contents/MacOS" || CheckLastResult
cp "IVPN Uninstaller-Info.plist" "bin/IVPN Uninstaller.app/Contents/Info.plist" || CheckLastResult

echo "[ ] IVPN Installer: Done"
echo "    ${_SCRIPT_DIR}/bin/IVPN Installer.app"
echo "    ${_SCRIPT_DIR}/bin/IVPN Uninstaller.app"

cd ${_BASE_DIR}
