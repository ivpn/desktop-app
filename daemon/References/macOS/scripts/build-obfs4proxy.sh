#!/bin/sh

# Exit immediately if a command exits with a non-zero status.
set -e

cd "$(dirname "$0")"
BASE_DIR="$(pwd)" #set base folder of script location

BUILD_DIR=${BASE_DIR}/../_deps/obfs4proxy_build # work directory
INSTALL_DIR=${BUILD_DIR}/../obfs4proxy_inst

echo "******** Creating work-folder (${BUILD_DIR})..."
rm -rf ${BUILD_DIR}
rm -rf ${INSTALL_DIR}
mkdir -pv ${BUILD_DIR}
mkdir -pv ${INSTALL_DIR}

echo "******** Cloning obfs4proxy sources..."
cd ${BUILD_DIR}
git clone https://github.com/Yawning/obfs4.git

echo "******** Compiling 'obfs4proxy'..."
cd obfs4
go build -o ${INSTALL_DIR}/obfs4proxy ./obfs4proxy

echo "********************************"
echo "******** BUILD COMPLETE ********"
echo "********************************"
