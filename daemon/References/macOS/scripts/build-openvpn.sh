#!/bin/sh

# ##############################################################################
# Define here OpenSSL and OpenVPN versions
# ##############################################################################
OPEN_SSL_VER=1.1.1k
OPEN_VPN_VER=v2.5.3

# This has to be installed
echo "******** Installing xcode command lines tools..."
xcode-select --install
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
mkdir -pv ${INSTALL_DIR}/include
mkdir -pv ${INSTALL_DIR}/lib

_ARCH="$( uname -m )"
echo "    ARCHITECTURE:            '${_ARCH}'"
if [ ${_ARCH} != "x86_64" ] && [ ${_ARCH} != "arm64" ]; then
  echo "ERROR: Unsupported architecture"
  exit 1
fi

echo "************************************************"
echo "******** Downloading OpenSSL sources..."
echo "************************************************"
cd ${BUILD_DIR}
curl https://www.openssl.org/source/openssl-${OPEN_SSL_VER}.tar.gz | tar zx

# ##############################################################################
# Compilation OpenSSl info:
# https://wiki.openssl.org/index.php/Compilation_and_Installation#OS_X
#
# If you want to use OS-default SHARED openssl libraries - skip steps of compilation OpenSSL
# ##############################################################################
echo "************************************************"
echo "******** Configuring OpenSSL..."
echo "************************************************"
cd ${BUILD_DIR}/openssl-${OPEN_SSL_VER}

if [ ${_ARCH} = "arm64" ]; then
  ./Configure darwin64-arm64-cc shared enable-ec_nistp_64_gcc_128 no-ssl2 no-ssl3 no-comp --openssldir=/usr/local/ssl/macos-arm64
else
  ./Configure darwin64-x86_64-cc shared enable-ec_nistp_64_gcc_128 no-ssl2 no-ssl3 no-comp --openssldir=/usr/local/ssl/macos-x86_64
fi

echo "************************************************"
echo "******** Compiling OpenSSL..."
echo "************************************************"
make

echo "************************************************"
echo "******** Copying OpenSSL include folder and static libraries..."

# if you want to use OS-default SHARED openssl libraries - not necessary to compile it.
# Just copy required headers of OpenSSL (include folder)
cp -r ${BUILD_DIR}/openssl-${OPEN_SSL_VER}/include/openssl ${INSTALL_DIR}/include/
# if you want to use OS-default SHARED openssl libraries - skip copying this static libraries
cp ${BUILD_DIR}/openssl-${OPEN_SSL_VER}/libcrypto.a ${INSTALL_DIR}/lib/
cp ${BUILD_DIR}/openssl-${OPEN_SSL_VER}/libssl.a ${INSTALL_DIR}/lib/

echo "************************************************"
echo "******** Downloading LZO sources..."
echo "************************************************"
cd ${BUILD_DIR}
curl http://www.oberhumer.com/opensource/lzo/download/lzo-2.08.tar.gz | tar zx
cd lzo-2.08

echo "************************************************"
echo "******** Compiling LZO..."
echo "************************************************"
CLFAGS="-mmacosx-version-min=10.6" ./configure --prefix="${INSTALL_DIR}" && make && make install

echo "************************************************"
echo "******** Cloning OpenVPN sources..."
echo "************************************************"
cd ${BUILD_DIR}
git clone https://github.com/OpenVPN/openvpn.git
cd openvpn

echo "************************************************"
echo "******** Checkout OpenVPN version (${OPEN_VPN_VER})..."
echo "************************************************"
git checkout ${OPEN_VPN_VER}

echo "************************************************"
echo "******** Patching OpenVPN..."
echo "************************************************"
patch -p2 < $BASE_DIR/patches/openvpn-osx-lion.patch

echo "************************************************"
echo "******** OpenVPN: Updating generated configuration files..."
echo "************************************************"
autoreconf -ivf

echo "************************************************"
echo "******** Configuring OpenVPN..."
echo "************************************************"
CFLAGS="-mmacosx-version-min=10.6 -I${INSTALL_DIR}/include" \
    LDFLAGS="-L${INSTALL_DIR}/lib" \
    ./configure --disable-debug --disable-server --enable-password-save

echo "************************************************"
echo "******** Compiling OpenVPN..."
echo "************************************************"
make

echo "********************************"
echo "******** BUILD COMPLETE ********"
echo "********************************"
mkdir -p ${INSTALL_DIR}/bin
cp ${BUILD_DIR}/openvpn/src/openvpn/openvpn ${INSTALL_DIR}/bin

set +e
${INSTALL_DIR}/bin/openvpn --version

echo "********************************"
echo "******** Please check the dynamic libraries OpenVPN uses ********"
echo "********************************"
otool -L ${INSTALL_DIR}/bin/openvpn

echo "********************************"
echo " DO NOT FORGET TO RECOMPILE 'IVPN Agent' project!"
echo "********************************"
