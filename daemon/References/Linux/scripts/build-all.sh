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

# check if we need to compile wireguard-tools
if [[ ! -f "../_deps/wireguard-tools_inst/wg-quick" ]] || [[ ! -f "../_deps/wireguard-tools_inst/wg" ]]
then
  echo "======================================================"
  echo "========== Compiling wireguard-tools ================="
  echo "======================================================"
  cd $SCRIPT_DIR
  ./build-wireguard-tools.sh
else
  echo "wireguard-tools already compiled. Skipping build."
fi

# check if we need to compile dnscrypt-proxy
if [[ ! -f "../_deps/dnscryptproxy_inst/dnscrypt-proxy" ]] 
then
  echo "======================================================"
  echo "========== Compiling dnscrypt-proxy =================="
  echo "======================================================"
  cd $SCRIPT_DIR
  ./build-dnscrypt-proxy.sh
else
  echo "dnscrypt-proxy already compiled. Skipping build."
fi

# check if we need to compile kem-helper
if [[ ! -f "../_deps/kem-helper/kem-helper-bin/kem-helper" ]]
then
  echo "======================================================"
  echo "========== Compiling kem-helper ======================"
  echo "======================================================"
  cd $SCRIPT_DIR
  ./build-kem-helper.sh
else
  echo "kem-helper already compiled. Skipping build."
fi

echo "======================================================"
echo "============ Compiling IVPN service =================="
echo "======================================================"
./build-daemon.sh $@

set +e
