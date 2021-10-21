#!/bin/sh

WG_TOOLS_VER=v1.0.20210914 # https://git.zx2c4.com/wireguard-tools/

# Exit immediately if a command exits with a non-zero status.
set -e

cd "$(dirname "$0")"
BASE_DIR="$(pwd)" #set base folder of script location

BUILD_DIR=${BASE_DIR}/../_deps/wireguard-tools_build # work directory
INSTALL_DIR=${BASE_DIR}/../_deps/wireguard-tools_inst

echo "******** Creating work-folder (${BUILD_DIR})..."
rm -rf ${BUILD_DIR}
rm -rf ${INSTALL_DIR}
mkdir -pv ${BUILD_DIR}
mkdir -pv ${INSTALL_DIR}

echo "******** Cloning wireguard-tools sources..."
cd ${BUILD_DIR}
git clone https://git.zx2c4.com/wireguard-tools/
cd wireguard-tools

echo "******** Checkout wireguard-tools version (${WG_TOOLS_VER})..."
git checkout ${WG_TOOLS_VER}
cd src

echo "******** Compiling 'wireguard-tools'..."
make

echo "******** Copying 'wireguard-tools' binaries..."
cp ${BUILD_DIR}/wireguard-tools/src/wg ${INSTALL_DIR}
cp ${BUILD_DIR}/wireguard-tools/src/wg-quick/linux.bash ${INSTALL_DIR}/wg-quick

echo "********************************"
echo "******** BUILD COMPLETE ********"
echo "********************************"