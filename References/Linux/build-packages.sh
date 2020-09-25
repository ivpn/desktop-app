#!/bin/sh

# Usage example:
#   build-packages.sh -v 0.0.1
#
#   NOTE! The correspond version of application already should be compiled (the file must be available: dist_electron/ivpn-ui-XXX.AppImage)

# To be able to build packages the 'fpm' tool shall be installed 
# (https://fpm.readthedocs.io/en/latest/installing.html)

# Useful commands (Ubuntu):
#
# To view *.deb package content:
#     dpkg -c ivpn_1.0_amd64.deb
# List of installet packets:
#     dpkg --list [<mask>]
# Install package:
#     apt-get install <pkg-name>
# Remove packet:
#     dpkg --remove <packetname>
# Remove (2):
#     apt-get remove ivpn
#     apt-get purge curl
#     apt-get autoremove
# Remove repository (https://www.ostechnix.com/how-to-delete-a-repository-and-gpg-key-in-ubuntu/):
#     add-apt-repository -r ppa:wireguard/wireguard
#     apt update
# List of services:
#     systemctl --type=service
# Start service:
#     systemctl start ivpn-service
# Remove BROKEN package (which is unable to uninstall by normal ways)
#     sudo mv /var/lib/dpkg/info/ivpn.* /tmp/
#     sudo dpkg --remove --force-remove-reinstreq ivpn

cd "$(dirname "$0")"

# check result of last executed command
CheckLastResult()
{
  if ! [ $? -eq 0 ]
  then #check result of last command
    if [ -n "$1" ]
    then
      echo $1
    else
      echo "FAILED"
    fi
    exit 1
  fi
}

SCRIPT_DIR="$( cd "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
OUT_DIR="$SCRIPT_DIR/_out_bin"

# ---------------------------------------------------------
# version info variables
VERSION=""

# reading version info from arguments
while getopts ":v:" opt; do
  case $opt in
    v) VERSION="$OPTARG"
    ;;
#    \?) echo "Invalid option -$OPTARG" >&2
#   ;;
  esac
done

if [ -z "$VERSION" ]
then
  echo "Usage:"
  echo "    $0 -v <version>"
  echo ""
  echo "    NOTE! The correspond version of application already should be compiled (the file must be available: dist_electron/ivpn-ui-XXX.AppImage)"
  exit 1 
fi

APP_BINARY="$SCRIPT_DIR/../../dist_electron/ivpn-ui-$VERSION.AppImage"
if [ -f $APP_BINARY ]; then
    echo "[ ] Integrating binary: $APP_BINARY"
else
  echo "[!] File not exists: '$APP_BINARY'"
  echo "    Plerase build IVPN UI project (do not forget to set correct version for it in 'package.json')"
  exit 1 
fi

echo "======================================================"
echo "============== Building packages ====================="
echo "======================================================"

set -e

TMPDIR="$SCRIPT_DIR/_tmp"
if [ -d "$TMPDIR" ]; then rm -Rf $TMPDIR; fi
mkdir -p $TMPDIR

CreatePackage()
{
  PKG_TYPE=$1
  EXTRA_ARGS=$2

  cd $TMPDIR
  
  # Scripts order is different for different types of packages
  # DEB Install:
  #   Clean             Upgrade
  #                     before_remove
  #   before_install    before_upgrade\before_install
  #                     after_remove
  #   after_install     after_upgrade\after_install
  #
  # DEB remove
  #   before_remove
  #   after_remove
  #
  # RPM Install:
  #   Clean             Upgrade
  #   before_install    before_upgrade\before_install 
  #   after_install     after_upgrade\after_install
  #                     before_remove
  #                     after_remove
  #
  # RPM remove
  #   before_remove
  #   after_remove
  #
  # NOTE! 'remove' scripts from old version!

  fpm $EXTRA_ARGS \
    --deb-no-default-config-files -s dir -t $PKG_TYPE -n ivpn-ui -v $VERSION --url https://www.ivpn.net --license "GNU GPL3" \
    --template-scripts --template-value pkg=$PKG_TYPE \
    --vendor "Privatus Limited" --maintainer "Privatus Limited" \
    --description "$(printf "UI client for IVPN service (https://www.ivpn.net)\nGraphical interface v$VERSION.")" \
    --before-install "$SCRIPT_DIR/package_scripts/before-install.sh" \
    --after-install "$SCRIPT_DIR/package_scripts/after-install.sh" \
    --before-remove "$SCRIPT_DIR/package_scripts/before-remove.sh" \
    --after-remove "$SCRIPT_DIR/package_scripts/after-remove.sh" \
    $SCRIPT_DIR/ui/IVPN.desktop=/opt/ivpn/ui/IVPN.desktop \
    $SCRIPT_DIR/ui/ivpnicon.svg=/opt/ivpn/ui/ivpnicon.svg \
    $APP_BINARY=/opt/ivpn/ui/ivpn-ui.AppImage
}

echo '---------------------------'
echo "DEB package..."
# to add dependency from another packet add extra arg "-d", example: "-d obfsproxy" 
CreatePackage "deb"

echo '---------------------------'
echo "RPM package..."
CreatePackage "rpm"

echo '---------------------------'
echo "Copying compiled pachages to '$OUT_DIR'..."
mkdir -p $OUT_DIR
yes | cp -f $TMPDIR/*.* $OUT_DIR

set +e