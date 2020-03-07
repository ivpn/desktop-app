#!/bin/sh

cd "$(dirname "$0")"

echo "======================================================"
echo "================ IVPN CLI ============================"
echo "======================================================"
cd ../../

VERSION=""
DATE="$(date "+%Y-%m-%d")"
COMMIT="$(git rev-list -1 HEAD)"

while getopts ":v:" opt; do
  case $opt in
    v) VERSION="$OPTARG"
    ;;
#    \?) echo "Invalid option -$OPTARG" >&2
#   ;;
  esac
done

echo "Version: $VERSION"
echo "Date   : $DATE"
echo "Commit : $COMMIT"

if [[ "$@" == *"-debug"* ]]
then
    echo "Compiling in DEBUG mode"
    go build -tags debug -o "ivpn" -ldflags "-X github.com/ivpn/desktop-app-daemon/version._version=$VERSION -X github.com/ivpn/desktop-app-daemon/version._commit=$COMMIT -X github.com/ivpn/desktop-app-daemon/version._time=$DATE"
else
    go build -o "ivpn" -ldflags "-X github.com/ivpn/desktop-app-daemon/version._version=$VERSION -X github.com/ivpn/desktop-app-daemon/version._commit=$COMMIT -X github.com/ivpn/desktop-app-daemon/version._time=$DATE"
fi


echo "Cpmpiled daemon binary: '$(pwd)/ivpn'"
