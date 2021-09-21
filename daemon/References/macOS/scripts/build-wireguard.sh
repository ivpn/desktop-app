#!/bin/sh

# ##############################################################################
# Define here WireGuard-Go version
# ##############################################################################
WG_GO_VER=0.0.20210424   # https://git.zx2c4.com/wireguard-go/
WG_TOOLS_VER=v1.0.20210914 # https://git.zx2c4.com/wireguard-tools/

# Exit immediately if a command exits with a non-zero status.
set -e

cd "$(dirname "$0")"
BASE_DIR="$(pwd)" #set base folder of script location

BUILD_DIR=${BASE_DIR}/../_deps/wg_build # work directory
INSTALL_DIR=${BUILD_DIR}/../wg_inst

echo "******** Creating work-folder (${BUILD_DIR})..."
rm -rf ${BUILD_DIR}
rm -rf ${INSTALL_DIR}
mkdir -pv ${BUILD_DIR}
mkdir -pv ${INSTALL_DIR}


echo "******** Cloning WireGuard-go sources..."
cd ${BUILD_DIR}
git clone https://git.zx2c4.com/wireguard-go/
cd wireguard-go
echo "******** Checkout wireguard-go version (${WG_GO_VER})..."
git checkout ${WG_GO_VER}
echo "******** Compiling 'wireguard-go'..."
CGO_CFLAGS=-mmacosx-version-min=10.10 CGO_LDFLAGS=-mmacosx-version-min=10.10 make

echo "******** Cloning wireguard-tools sources..."
cd ${BUILD_DIR}
git clone https://git.zx2c4.com/wireguard-tools/
cd wireguard-tools/src
echo "******** Checkout wireguard-tools version (${WG_TOOLS_VER})..."
git checkout ${WG_TOOLS_VER}
echo "******** Compiling 'wireguard-tools'..."
CFLAGS=-mmacosx-version-min=10.10 LDFLAGS=-mmacosx-version-min=10.10 make

echo "********************************"
echo "******** BUILD COMPLETE ********"
echo "********************************"

echo "******** Copying compiled binaries to '$BUILD_DIR"
cd ${BUILD_DIR}
cp ./wireguard-go/wireguard-go $INSTALL_DIR
cp ./wireguard-tools/src/wg $INSTALL_DIR
