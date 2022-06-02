#!/bin/sh

# Usage example:
#   build-packages.sh -v 0.0.1
#

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

ARCH="$( node -e 'console.log(process.arch)' )"
SCRIPT_DIR="$( cd "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
OUT_DIR="$SCRIPT_DIR/_out_bin"
APP_UNPACKED_DIR="$SCRIPT_DIR/../../dist_electron/linux-unpacked"
APP_UNPACKED_DIR_ARCH="$SCRIPT_DIR/../../dist_electron/linux-${ARCH}-unpacked"
APP_BIN_DIR="$SCRIPT_DIR/../../dist_electron/bin"
IVPN_DESKTOP_UI2_SOURCES="$SCRIPT_DIR/../../"

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
  exit 1
fi

echo "Architecture: $ARCH"
echo "======================================================"
echo "============ Building UI binary ======================"
echo "======================================================"

if [ -d $APP_UNPACKED_DIR ]; then
  echo "[+] Removing: $APP_UNPACKED_DIR"
  rm -fr "$APP_UNPACKED_DIR"
fi
if [ -d $APP_UNPACKED_DIR_ARCH ]; then
  echo "[+] Removing: $APP_UNPACKED_DIR_ARCH"
  rm -fr "$APP_UNPACKED_DIR_ARCH"
fi

if [ -d $APP_BIN_DIR ]; then
  echo "[+] Removing: $APP_BIN_DIR"
  rm -fr "$APP_BIN_DIR"
fi

cat "$IVPN_DESKTOP_UI2_SOURCES/package.json" | grep \"version\" | grep \"$VERSION\"
CheckLastResult "ERROR: Please set correct version in file '${IVPN_DESKTOP_UI2_SOURCES}package.json'"

echo "*** Installing NPM molules ... ***"
cd $IVPN_DESKTOP_UI2_SOURCES
CheckLastResult
npm install
CheckLastResult
cd $SCRIPT_DIR
CheckLastResult

echo "*** Building Electron app ... ***"
$SCRIPT_DIR/compile-ui.sh
CheckLastResult

if [ -d $APP_UNPACKED_DIR_ARCH ]; then
    # for non-standard architecture we must use the architecture-dependend path
    echo "Info: Non 'default' architecture!" 
    APP_UNPACKED_DIR=$APP_UNPACKED_DIR_ARCH
fi
if [ -d $APP_UNPACKED_DIR ]; then
    echo "[ ] Exist: $APP_UNPACKED_DIR"
else
  echo "[!] Folder not exists: '$APP_UNPACKED_DIR'"
  echo "    Build IVPN UI project (do not forget to set correct version for it in 'package.json')"
  exit 1
fi
if [ -f "$APP_UNPACKED_DIR/ivpn-ui" ]; then
    echo "[ ] Exist: $APP_UNPACKED_DIR/ivpn-ui"
else
  echo "[!] File not exists: '$APP_UNPACKED_DIR/ivpn-ui'"
  echo "    Build IVPN UI project (do not forget to set correct version for it in 'package.json')"
  exit 1
fi

echo "[ ] Renaming: '$APP_UNPACKED_DIR' -> '$APP_BIN_DIR'"
mv $APP_UNPACKED_DIR $APP_BIN_DIR
CheckLastResult

if [ ! -z "$SNAPCRAFT_BUILD_ENVIRONMENT" ]; then
    echo "! SNAPCRAFT_BUILD_ENVIRONMENT detected !"
    echo "! DEB/RPM packages build skipped !"
    exit 0
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
  #   (On Install)      (On Upgrade)
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
  #   (On Install)      (On Upgrade)
  #   before_install    before_upgrade\before_install
  #   after_install     after_upgrade\after_install
  #                     before_remove
  #                     after_remove
  #
  # RPM remove
  #   before_remove
  #   after_remove
  #
  # NOTE! 'remove' scripts is using from old version!

  fpm -d ivpn $EXTRA_ARGS \
    --rpm-rpmbuild-define "_build_id_links none" \
    --deb-no-default-config-files -s dir -t $PKG_TYPE -n ivpn-ui -v $VERSION --url https://www.ivpn.net --license "GNU GPL3" \
    --template-scripts --template-value pkg=$PKG_TYPE --template-value version=$VERSION \
    --vendor "Privatus Limited" --maintainer "Privatus Limited" \
    --description "$(printf "UI client for IVPN service (https://www.ivpn.net)\nGraphical interface v$VERSION.")" \
    --before-install "$SCRIPT_DIR/package_scripts/before-install.sh" \
    --after-install "$SCRIPT_DIR/package_scripts/after-install.sh" \
    --before-remove "$SCRIPT_DIR/package_scripts/before-remove.sh" \
    --after-remove "$SCRIPT_DIR/package_scripts/after-remove.sh" \
    $SCRIPT_DIR/ui/IVPN.desktop=/opt/ivpn/ui/IVPN.desktop \
    $SCRIPT_DIR/ui/ivpnicon.svg=/opt/ivpn/ui/ivpnicon.svg \
    $APP_BIN_DIR=/opt/ivpn/ui/
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
