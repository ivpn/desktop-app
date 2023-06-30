#!/bin/sh

V2RAY_VER=v5.7.0

# Exit immediately if a command exits with a non-zero status.
set -e

cd "$(dirname "$0")"
BASE_DIR="$(pwd)" #set base folder of script location

BUILD_DIR=${BASE_DIR}/../_deps/v2ray_build # work directory
INSTALL_DIR=${BUILD_DIR}/../v2ray_inst

echo "******** Creating work-folder (${BUILD_DIR})..."
rm -rf ${BUILD_DIR}
rm -rf ${INSTALL_DIR}
mkdir -pv ${BUILD_DIR}
mkdir -pv ${INSTALL_DIR}

echo "******** Cloning V2Ray sources..."
cd ${BUILD_DIR}
git clone  --depth 1 --branch ${V2RAY_VER} https://github.com/v2fly/v2ray-core.git
cd v2ray-core/main

echo "******** Compiling 'V2Ray'..."
go build -o ${INSTALL_DIR}/v2ray -trimpath -ldflags "-s -w"

echo "********************************"
echo "******** BUILD COMPLETE ********"
echo "********************************"