#!/bin/bash

cd "$(dirname "$0")"

SCRIPT_DIR="$( cd "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
OUT_DIR="$SCRIPT_DIR/_out_bin"
OUT_FILE="$OUT_DIR/ivpn"
OUT_BASH_COMPLETION_SCRIPT=$OUT_DIR/ivpn.bash-completion

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

echo "======================================================"
echo "============== Compiling IVPN CLI ===================="
echo "======================================================"
echo "Version: $VERSION"
echo "Date   : $DATE"
echo "Commit : $COMMIT"

cd $SCRIPT_DIR/../../

echo "* updating dependencies..."
go get -v

if [[ "$@" == *"-debug"* ]]
then
    echo "Compiling in DEBUG mode"
    CGO_ENABLED=0 go build -tags debug -o "$OUT_FILE" -trimpath -ldflags "-X github.com/ivpn/desktop-app/daemon/version._version=$VERSION -X github.com/ivpn/desktop-app/daemon/version._commit=$COMMIT -X github.com/ivpn/desktop-app/daemon/version._time=$DATE"
else
    CGO_ENABLED=0 go build -o "$OUT_FILE" -trimpath -ldflags "-s -w -X github.com/ivpn/desktop-app/daemon/version._version=$VERSION -X github.com/ivpn/desktop-app/daemon/version._commit=$COMMIT -X github.com/ivpn/desktop-app/daemon/version._time=$DATE"
fi

# generate bash-completion script
$SCRIPT_DIR/bash-completion-generator-ivpn-cli.sh "$OUT_FILE" > "$OUT_BASH_COMPLETION_SCRIPT"
bash -n "$OUT_BASH_COMPLETION_SCRIPT" # check bash-completion script syntax


echo "Compiled CLI binary   : '$OUT_FILE'"
echo "Bash-completion script: '$OUT_BASH_COMPLETION_SCRIPT'"

set +e
