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

# Build
cd $SCRIPT_DIR/../../../

BUILDTAG_DEBUG=""  # "debug"
BUILDTAG_NOWIFI="" # "nowifi"

if [[ "$@" == *"-debug"* ]]
then
  echo "[!] Compiling in DEBUG mode."
  BUILDTAG_DEBUG="debug"
fi
if [ ! -z "$IVPN_NO_WIFI" ]; then
  echo "[!] WIFI functionality DISABLED."
  BUILDTAG_NOWIFI="nowifi"
fi

CGO_ENABLED=0 go build -buildmode=pie -tags "${BUILDTAG_DEBUG} ${BUILDTAG_NOWIFI}" -o "$OUT_FILE" -trimpath -ldflags "-X github.com/ivpn/desktop-app/daemon/version._version=$VERSION -X github.com/ivpn/desktop-app/daemon/version._commit=$COMMIT -X github.com/ivpn/desktop-app/daemon/version._time=$DATE"

echo "Compiled binary: '$OUT_FILE'"

set +e
