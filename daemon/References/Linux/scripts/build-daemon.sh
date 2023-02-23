#!/bin/bash

cd "$(dirname "$0")"

SCRIPT_DIR="$( cd "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
OUT_DIR="$SCRIPT_DIR/_out_bin"
OUT_FILE="$OUT_DIR/ivpn-service"

set -e

# make output dir if not exists
mkdir -p $OUT_DIR

# version info variables
VERSION=""
DATE="$(date "+%Y-%m-%d")"
COMMIT=""

# reading version info from arguments
while getopts ":v:c:" opt; do
  case $opt in
    v) VERSION="$OPTARG"
    ;;
    c) COMMIT="$OPTARG"
    ;;
#    \?) echo "Invalid option -$OPTARG" >&2
#   ;;
  esac
done

if [ -z "$COMMIT" ]; then
  COMMIT="$(git rev-list -1 HEAD)"
fi

# updating servers.json
cd $SCRIPT_DIR
./update-servers.sh

echo "!!!!!!!!!!!!!!!!!!!! INFO !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
echo "Version: $VERSION"
echo "Date   : $DATE"
echo "Commit : $COMMIT"
echo "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"

# Check required GLIBC version. 
# Compiling with the new GLIBC version will not allow the program to start on systems with the old GLIBC (error example: "version 'GLIBC_2.34' not found"). 
# Useful links:
#   https://utcc.utoronto.ca/~cks/space/blog/programming/GoAndGlibcVersioning
# Useful commands:
#   ldd -r -v <binary_file> # check shared libraries dependencies
#
# TODO: think how to avoid using CGO
if [ ! -z "$IVPN_BUILD_SKIP_GLIBC_VER_CHECK" ] || [ ! -z "$GITHUB_ACTIONS" ]; 
then
  echo "[!] ! GLIBC version check skipped (according to env vars configuration) !"
else
  GLIBC_VER_MAX_REQUIRED="2.31"
  GLIBC_VER=$(ldd --version | grep "ldd (" | awk '{print $(NF)}')
  if [[ "${GLIBC_VER}" > "${GLIBC_VER_MAX_REQUIRED}" ]]; 
  then
      echo "[!] GLIBC version '${GLIBC_VER}' is greater than reqired '${GLIBC_VER_MAX_REQUIRED}'"
      echo "[!] Compiling with the new GLIBC version will not allow the program to start on systems with the old GLIBC."
      echo "[ ] (you can define env var 'IVPN_BUILD_SKIP_GLIBC_VER_CHECK' to skip this check"
      read -p "[?] Do you want to continue? [y\n] (N - default): " yn
      case $yn in
        [Yy]* ) ;;
        * ) 
        echo "[!] Build interrupted by user"
        exit 1
        ;;
      esac
  fi
fi

# Build
cd $SCRIPT_DIR/../../../

if [[ "$@" == *"-debug"* ]]
then
    echo "Compiling in DEBUG mode"
    go build -buildmode=pie -tags debug -o "$OUT_FILE" -trimpath -ldflags "-X github.com/ivpn/desktop-app/daemon/version._version=$VERSION -X github.com/ivpn/desktop-app/daemon/version._commit=$COMMIT -X github.com/ivpn/desktop-app/daemon/version._time=$DATE"
else
    go build -buildmode=pie -o "$OUT_FILE" -trimpath -ldflags "-s -w -X github.com/ivpn/desktop-app/daemon/version._version=$VERSION -X github.com/ivpn/desktop-app/daemon/version._commit=$COMMIT -X github.com/ivpn/desktop-app/daemon/version._time=$DATE"
fi

echo "Compiled binary: '$OUT_FILE'"

set +e
