#!/bin/sh

# ##############################################################################
# Define here WireGuard-Go version
# ##############################################################################
WG_GO_VER=0.0.20230223      # https://git.zx2c4.com/wireguard-go/
WG_TOOLS_VER=v1.0.20210914  # https://git.zx2c4.com/wireguard-tools/

# Exit immediately if a command exits with a non-zero status.
set -e

cd "$(dirname "$0")"
BASE_DIR="$(pwd)" #set base folder of script location

BUILD_DIR=${BASE_DIR}/../_deps/wg_build # work directory
INSTALL_DIR=${BUILD_DIR}/../wg_inst

# Function to set up temporary Go environment
# It downloads the specified Go version and sets up the environment variables
# Arguments:
#   $1: Go version to download
setup_go_env() {
    local GO_VERSION=$1
    local TEMP_GOROOT="${BUILD_DIR}/go-${GO_VERSION}"
    local TEMP_GOPATH="${BUILD_DIR}/gopath"
    
    # Get system architecture
    local ARCH="$(uname -m)"
    if [ "${ARCH}" != "x86_64" ] && [ "${ARCH}" != "arm64" ]; then
        echo "ERROR: Unsupported architecture: ${ARCH}"
        exit 1
    fi
    
    # Create GOPATH directory if it doesn't exist
    mkdir -p "${TEMP_GOPATH}"
    
    # Check if Go is already installed in the expected location
    local NEED_DOWNLOAD=true
    if [ -d "${TEMP_GOROOT}" ] && [ -f "${TEMP_GOROOT}/bin/go" ]; then
        NEED_DOWNLOAD=false
    fi
    
    if [ "$NEED_DOWNLOAD" = true ]; then
        echo "Downloading Go ${GO_VERSION} for architecture ${ARCH}..."
        mkdir -p "${TEMP_GOROOT}"
        curl -sSL "https://go.dev/dl/go${GO_VERSION}.darwin-${ARCH}.tar.gz" | tar -xz -C "${TEMP_GOROOT}" --strip-components=1
    fi

    # Use the temporary Go installation
    export PATH="${TEMP_GOROOT}/bin:$PATH"
    export GOROOT="${TEMP_GOROOT}"
    export GOPATH="${TEMP_GOPATH}"
    
    # Verify Go installation
    echo "Verifying Go installation..."
    go version
    if [ $? -ne 0 ]; then
        echo "Failed to set up Go environment"
        exit 1
    fi
}

echo "******** Creating work-folder (${BUILD_DIR})..."

if [ -d "${BUILD_DIR}" ]; then
  # Ensure the build directory is writable, as Go makes files in the module cache read-only
  chmod -R +w "${BUILD_DIR}"
fi
rm -rf ${BUILD_DIR}
rm -rf ${INSTALL_DIR}
mkdir -pv ${BUILD_DIR}
mkdir -pv ${INSTALL_DIR}

echo "******** Setting up Go environment version ${GO_VERSION}..."
# Use the temporary Go v1.22.12 environment because 'wireguard-go' fails when using Go >= 1.23
setup_go_env "1.22.12" # TODO: Remove this when wireguard-go supports latest Go versions

echo "******** Cloning WireGuard-go sources (version ${WG_GO_VER})..."
cd ${BUILD_DIR}
git clone --branch "${WG_GO_VER}" --depth 1 https://git.zx2c4.com/wireguard-go/
cd wireguard-go

echo "******** Compiling 'wireguard-go'..."
CGO_ENABLED=1 CGO_CFLAGS=-mmacosx-version-min=10.14 CGO_LDFLAGS=-mmacosx-version-min=10.14 make

echo "******** Cloning wireguard-tools sources (version ${WG_TOOLS_VER})..."
cd ${BUILD_DIR}
git clone --branch "${WG_TOOLS_VER}" --depth 1 https://git.zx2c4.com/wireguard-tools/
cd wireguard-tools/src

echo "******** Compiling 'wireguard-tools'..."
CFLAGS=-mmacosx-version-min=10.14 LDFLAGS=-mmacosx-version-min=10.14 make

echo "********************************"
echo "******** BUILD COMPLETE ********"
echo "********************************"

echo "******** Copying compiled binaries to '$BUILD_DIR"
cd ${BUILD_DIR}
cp ./wireguard-go/wireguard-go $INSTALL_DIR
cp ./wireguard-tools/src/wg $INSTALL_DIR
