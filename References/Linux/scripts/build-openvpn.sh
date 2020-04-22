#!/bin/sh

cd "$(dirname "$0")"

OPEN_VPN_VER=2.4.8

# There are some dependencies required to build OpenVPN
# Here is a commands to install required packages foe Ubuntu:
#
#   sudo apt-get update -y
# LibSSl headers:
#   sudo apt-get install -y libssl-dev
#
#   sudo apt-get install liblz4-dev
# If command 'route' not found, but can be installed with:
#   sudo apt install net-tools
# If configure: error: lzo enabled but missing
#   sudo apt-get install liblzo2-dev libpam0g-dev

# Exit immediately if a command exits with a non-zero status.
set -e

cd "$(dirname "$0")"
BASE_DIR="$(pwd)" #set base folder of script location

BUILD_DIR=${BASE_DIR}/../_deps/openvpn_build # work directory
INSTALL_DIR=${BUILD_DIR}/../openvpn_inst
echo "******** Creating work-folder (${BUILD_DIR})..."
rm -rf ${BUILD_DIR}
rm -rf ${INSTALL_DIR}

mkdir -pv ${BUILD_DIR}
mkdir -pv ${INSTALL_DIR}


echo "******** Obtaining OpenVPN sources (v${OPEN_VPN_VER})..."
cd ${BUILD_DIR}
wget https://swupdate.openvpn.org/community/releases/openvpn-$OPEN_VPN_VER.tar.gz
tar -zxf openvpn-$OPEN_VPN_VER.tar.gz
cd openvpn-$OPEN_VPN_VER

echo "******** Building..."
./configure --prefix=$INSTALL_DIR
make
make install

echo "********"
echo "DONE. Binary location: ${INSTALL_DIR}/sbin"
