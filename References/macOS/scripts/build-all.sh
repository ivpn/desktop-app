#!/bin/sh

cd "$(dirname "$0")"

echo "############################################"
echo "### Building IVPN Daemon"
echo "### OpenVPN and WireGuard will be also recompiled if they are not exists"

if [ "$#" -eq 0 ]
then
  echo "### Possible arguments:"
  echo "###   -norebuild    - do not rebuild openVPN and WireGuard binaries is they already compiled"
  echo "###   -debug        - compile IVPN Daemon in debug mode"
fi
echo "############################################"

if [[ ! -f "../_deps/openvpn_inst/bin/openvpn" ]] || [[ ! -f "../_deps/wg_inst/wg" ]] || [[ ! -f "../_deps/wg_inst/wireguard-go" ]]
then
  echo "Please, check/modify required versions at the begining of scripts:"
  echo "    build-openvpn.sh"
  echo "    build-wireguard.sh"

  read -p "Press enter to start ..."
fi

# Exit immediately if a command exits with a non-zero status.
set -e

function BuildOpenVPN
{
  echo "############################################"
  echo "### OpenVPN"
  echo "############################################"
  ./build-openvpn.sh
}

function BuildWireGuard
{
  echo "############################################"
  echo "### WireGuard"
  echo "############################################"
  ./build-wireguard.sh
}

if [[ "$@" == *"-norebuild"* ]]
then
    # check if we need to compile openvpn
    if [[ ! -f "../_deps/openvpn_inst/bin/openvpn" ]]
    then
      echo "OpenVPN not compiled"
      BuildOpenVPN
    else
      echo "OpenVPN already compiled. Skipping build."
    fi

    # check if we need to compile WireGuard
    if [[ ! -f "../_deps/wg_inst/wg" ]] || [[ ! -f "../_deps/wg_inst/wireguard-go" ]]
    then
      echo "WireGuard not compiled"
      BuildWireGuard
    else
      echo "WireGuard already compiled. Skipping build."
    fi

else
  # recompile openvpn and WireGuard
  BuildOpenVPN
  BuildWireGuard
fi

# updating servers.json
./update-servers.sh

echo "======================================================"
echo "=============== IVPN Agent ==========================="
echo "======================================================"
cd ../../../

if [[ "$@" == *"-debug"* ]]
then
    echo "Compiling in DEBUG mode"
    go build -tags debug -o "IVPN Agent"
else
    go build -o "IVPN Agent"
fi


echo "Cpmpiled daemon binary: '$(pwd)/IVPN Agent'"
