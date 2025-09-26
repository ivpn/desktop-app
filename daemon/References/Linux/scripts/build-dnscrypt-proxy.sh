#!/bin/sh

# TODO: define here dnscrypt-proxy version to build
_VERSION=2.1.14

# Exit immediately if a command exits with a non-zero status.
set -e

cd "$(dirname "$0")"
BASE_DIR="$(pwd)" #set base folder of script location

BUILD_DIR=${BASE_DIR}/../_deps/dnscryptproxy_build # work directory
INSTALL_DIR=${BUILD_DIR}/../dnscryptproxy_inst

echo "******** Creating work-folder (${BUILD_DIR})..."
rm -rf ${BUILD_DIR}
rm -rf ${INSTALL_DIR}
mkdir -pv ${BUILD_DIR}
mkdir -pv ${INSTALL_DIR}

echo "******** Cloning sources 'dnscrypt-proxy'..."
cd ${BUILD_DIR}
git clone https://github.com/DNSCrypt/dnscrypt-proxy.git
cd dnscrypt-proxy

echo "******** Checkout version ${_VERSION} of 'dnscrypt-proxy'..."
git checkout tags/${_VERSION}

echo "******** Compiling 'dnscrypt-proxy'..."
go build -o ${INSTALL_DIR}/dnscrypt-proxy -trimpath -ldflags "-s -w" ./dnscrypt-proxy

echo "******** BUILD COMPLETE ********"