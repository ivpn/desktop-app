#!/bin/sh

# NOTE!!! Special build script for AUR repository:
# Version v5.7.0 (which is the latest at the current moment) of V2Ray cannot be compiled with Go 1.21 yet.
# Therefore, we are using the latest V2Ray sources available at the time of writing this script. 
# These sources have not been updated to a new version but can be compiled with Go 1.21.

# V2RAY_VER=v5.7.0

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
#git clone  --depth 1 --branch ${V2RAY_VER} https://github.com/v2fly/v2ray-core.git
# cd v2ray-core/main
git clone --shallow-since="2023-08-30" https://github.com/v2fly/v2ray-core.git
cd v2ray-core
git checkout 87155bfc7193efd348db05df7c31e6ae71d890fe # Commit: Aug 31, 2023 (latest commit on a moment writing this line)
cd main

echo "******** Compiling 'V2Ray'..."
go build -o ${INSTALL_DIR}/v2ray -trimpath -ldflags "-s -w"

echo "********************************"
echo "******** BUILD COMPLETE ********"
echo "********************************"