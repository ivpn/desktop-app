#!/bin/sh

cd "$(dirname "$0")"

SCRIPT_DIR="$( cd "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
OUT_DIR="$SCRIPT_DIR/_out_bin"

# ====== Architecture setup ======
_HOST_ARCH="$(uname -m)"
ARCH_TARGET="${ARCH_TARGET:-$_HOST_ARCH}"
case "$ARCH_TARGET" in
  arm64)  _GOARCH="arm64" ;;
  x86_64) _GOARCH="amd64" ;;
  *)
    echo "ERROR: Unsupported ARCH_TARGET='$ARCH_TARGET'. Use 'arm64' or 'x86_64'."
    exit 1
    ;;
esac
echo "    ARCH_TARGET: ${ARCH_TARGET}"
# ====== End architecture setup ======

OUT_FILE="$OUT_DIR/${ARCH_TARGET}/ivpn"

set -e

# make output dir if not exists
mkdir -p "$OUT_DIR/${ARCH_TARGET}"

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

echo "======================================================"
echo "============== Compiling IVPN CLI ===================="
echo "======================================================"
echo "Version: $VERSION"
echo "Date   : $DATE"
echo "Commit : $COMMIT"

cd $SCRIPT_DIR/../../

# CGO_ENABLED=0: the CLI needs no native code, and both architectures are
# built the same way regardless of the host.
if [[ "$@" == *"-debug"* ]]
then
    echo "Compiling in DEBUG mode"
    GOOS=darwin GOARCH=${_GOARCH} CGO_ENABLED=0 go build -tags debug -o "$OUT_FILE" -trimpath -ldflags "-X github.com/ivpn/desktop-app/daemon/version._version=$VERSION -X github.com/ivpn/desktop-app/daemon/version._commit=$COMMIT -X github.com/ivpn/desktop-app/daemon/version._time=$DATE"
else
    GOOS=darwin GOARCH=${_GOARCH} CGO_ENABLED=0 go build -o "$OUT_FILE" -trimpath -ldflags "-s -w -X github.com/ivpn/desktop-app/daemon/version._version=$VERSION -X github.com/ivpn/desktop-app/daemon/version._commit=$COMMIT -X github.com/ivpn/desktop-app/daemon/version._time=$DATE"
fi

echo "Compiled CLI binary: '$OUT_FILE'"

set +e
