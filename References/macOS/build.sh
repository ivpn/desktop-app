#!/bin/sh

cd "$(dirname "$0")"

echo "======================================================"
echo "================ IVPN CLI ============================"
echo "======================================================"
cd ../../

if [[ "$@" == *"-debug"* ]]
then
    echo "Compiling in DEBUG mode"
    go build -tags debug -o "ivpn"
else
    go build -o "ivpn"
fi


echo "Cpmpiled daemon binary: '$(pwd)/ivpn'"
