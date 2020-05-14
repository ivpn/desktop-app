#!/bin/bash

# To install OpenVPN from repository on Ubuntu:
#   sudo apt install openvpn
# To install WireGuard from repository on Ubuntu:
#   Ubuntu ≥ 19.10:
#       sudo apt install wireguard
#   Ubuntu ≤ 19.04:
#       sudo add-apt-repository ppa:wireguard/wireguard
#       sudo apt-get update
#       sudo apt-get install wireguard

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
COMMIT="$(git rev-list -1 HEAD)"

# reading version info from arguments
while getopts ":v:" opt; do
  case $opt in
    v) VERSION="$OPTARG"
    ;;
#    \?) echo "Invalid option -$OPTARG" >&2
#   ;;
  esac
done

# updating servers.json
./update-servers.sh

echo "======================================================"
echo "============ Compiling IVPN service =================="
echo "======================================================"
echo "Version: $VERSION"
echo "Date   : $DATE"
echo "Commit : $COMMIT"

cd $SCRIPT_DIR/../../../

if [[ "$@" == *"-debug"* ]]
then
    echo "Compiling in DEBUG mode"
    go build -tags debug -o "$OUT_FILE" -ldflags "-X github.com/ivpn/desktop-app-daemon/version._version=$VERSION -X github.com/ivpn/desktop-app-daemon/version._commit=$COMMIT -X github.com/ivpn/desktop-app-daemon/version._time=$DATE"
else
    go build -o "$OUT_FILE" -ldflags "-X github.com/ivpn/desktop-app-daemon/version._version=$VERSION -X github.com/ivpn/desktop-app-daemon/version._commit=$COMMIT -X github.com/ivpn/desktop-app-daemon/version._time=$DATE"
fi

echo "Compiled binary: '$OUT_FILE'"

set +e
