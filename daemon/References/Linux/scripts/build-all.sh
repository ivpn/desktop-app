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

echo "!!!!!!!!!!!!!!!!!!!! INFO !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
echo "Version: $VERSION"
echo "Date   : $DATE"
echo "Commit : $COMMIT"
echo "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"

# check if we need to compile obfs4proxy
if [[ ! -f "../_deps/obfs4proxy_inst/obfs4proxy" ]]
then
  echo "======================================================"
  echo "========== Compiling obfs4proxy ======================"
  echo "======================================================"
  cd $SCRIPT_DIR
  ./build-obfs4proxy.sh
else
  echo "obfs4proxy already compiled. Skipping build."
fi

# updating servers.json
cd $SCRIPT_DIR
./update-servers.sh

echo "======================================================"
echo "============ Compiling IVPN service =================="
echo "======================================================"

cd $SCRIPT_DIR/../../../

if [[ "$@" == *"-debug"* ]]
then
    echo "Compiling in DEBUG mode"
    go build -tags debug -o "$OUT_FILE" -trimpath -ldflags "-X github.com/ivpn/desktop-app/daemon/version._version=$VERSION -X github.com/ivpn/desktop-app/daemon/version._commit=$COMMIT -X github.com/ivpn/desktop-app/daemon/version._time=$DATE"
else
    go build -o "$OUT_FILE" -trimpath -ldflags "-X github.com/ivpn/desktop-app/daemon/version._version=$VERSION -X github.com/ivpn/desktop-app/daemon/version._commit=$COMMIT -X github.com/ivpn/desktop-app/daemon/version._time=$DATE"
fi

echo "Compiled binary: '$OUT_FILE'"

set +e
