#!/bin/bash

cd "$(dirname "$0")"

SCRIPT_DIR="$( cd "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
OUT_DIR="$SCRIPT_DIR/_out_bin"

GLIBC_VER_MAX_REQUIRED="2.27"

set -e

# make output dir if not exists
mkdir -p $OUT_DIR

# Check required GLIBC version. 
# Compiling with the new GLIBC version will not allow the program to start on systems with the old GLIBC (error example: "version 'GLIBC_2.34' not found"). 
# Useful links:
#   https://utcc.utoronto.ca/~cks/space/blog/programming/GoAndGlibcVersioning
# Useful commands:
#   ldd -r -v <binary_file> # check shared libraries dependencies
#
if [ ! -z "$IVPN_BUILD_SKIP_GLIBC_VER_CHECK" ] || [ ! -z "$GITHUB_ACTIONS" ]; 
then
  echo "[!] ! GLIBC version check skipped (according to env vars configuration) !"
else  
  GLIBC_VER=$(ldd --version | grep "ldd (" | awk '{print $(NF)}')
  if [[ "${GLIBC_VER}" > "${GLIBC_VER_MAX_REQUIRED}" ]]; 
  then
      echo "[!] GLIBC version '${GLIBC_VER}' is newer than required '${GLIBC_VER_MAX_REQUIRED}'"
      echo "[!]     Binaries compiled with newer GLIBC will not run on systems with older GLIBC versions."
      echo "[!]     e.g. this affects: 'wg', 'obfs4proxy', 'dnscrypt-proxy', 'v2ray', and 'kem-helper' binaries"
      echo "[ ]     (to skip this check, set environment variable: IVPN_BUILD_SKIP_GLIBC_VER_CHECK=1)"
      read -p "[?] Continue with potentially incompatible build? [y/N]: " yn
      case $yn in
        [Yy]* ) ;;
        * ) 
        echo "[!] Build interrupted by user"
        exit 1
        ;;
      esac
  fi
fi

echo ""

# check if we need to compile obfs4proxy
if [[ ! -f "../_deps/obfs4proxy_inst/obfs4proxy" ]]
then
  echo "======================================================"
  echo "========== Compiling obfs4proxy ======================"
  echo "======================================================"
  cd $SCRIPT_DIR
  ./build-obfs4proxy.sh
else
  echo " - 'obfs4proxy' already compiled. Skipping build."
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
  echo " - 'wireguard-tools' already compiled. Skipping build."
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
  echo " - 'dnscrypt-proxy' already compiled. Skipping build."
fi

# check if we need to compile v2ray
if [[ ! -f "../_deps/v2ray_inst/v2ray" ]]
then
  echo "======================================================"
  echo "========== Compiling v2ray ==========================="
  echo "======================================================"
  cd $SCRIPT_DIR

  if [ ! -z "$GITHUB_ACTIONS" ]; 
  then
    echo "! GITHUB_ACTIONS detected ! It is just a build test."
    echo "! Skipped compilation of V2Ray !"
  else
    ./build-v2ray.sh
  fi

else
  echo " - 'v2ray' already compiled. Skipping build."
fi

# check if we need to compile kem-helper
if [[ ! -f "../_deps/kem-helper/kem-helper-bin/kem-helper" ]]
then
  echo "======================================================"
  echo "========== Compiling kem-helper ======================"
  echo "======================================================"
  cd $SCRIPT_DIR

  if [ ! -z "$GITHUB_ACTIONS" ]; 
  then
    echo "! GITHUB_ACTIONS detected ! It is just a build test."
    echo "! Skipped compilation of kem-helper !"
  else
    ./build-kem-helper.sh
  fi

else
  echo " - 'kem-helper' already compiled. Skipping build."
fi

echo "======================================================"
echo "============ Compiling IVPN service =================="
echo "======================================================"
./build-daemon.sh $@

echo "======================================================"
echo "[+] Checking GLIBC version required by compiled binaries ..."
BINARIES=(
    "$SCRIPT_DIR/../_deps/wireguard-tools_inst/wg"
    "$SCRIPT_DIR/../_deps/obfs4proxy_inst/obfs4proxy" 
    "$SCRIPT_DIR/../_deps/dnscryptproxy_inst/dnscrypt-proxy"
    "$SCRIPT_DIR/../_deps/v2ray_inst/v2ray"
    "$SCRIPT_DIR/../_deps/kem-helper/kem-helper-bin/kem-helper"
    "$SCRIPT_DIR/_out_bin/ivpn-service"
)
ISSUES=0
for bin in "${BINARIES[@]}"; do
    [[ ! -f "$bin" ]] && { echo "    [SKIP] $(basename "$bin") - not found"; continue; }
    if ! ldd "$bin" &>/dev/null; then
        echo "    [OK] $(basename "$bin") - static binary"
    else
        MAX_GLIBC=$(objdump -T "$bin" 2>/dev/null | grep -o 'GLIBC_[0-9.]*' | sort -V | tail -1 | cut -d_ -f2)
        if [[ -n "$MAX_GLIBC" && "$MAX_GLIBC" > "$GLIBC_VER_MAX_REQUIRED" ]]; then
            echo "    [FAIL] $(basename "$bin") - requires GLIBC_$MAX_GLIBC > $GLIBC_VER_MAX_REQUIRED"
            ISSUES=$((ISSUES + 1))
        else
            echo "    [OK] $(basename "$bin") - compatible (GLIBC_${MAX_GLIBC:-none})"
        fi
    fi
done
if [[ $ISSUES -eq 0 ]]; then
    echo "✓ All binaries compatible with GLIBC $GLIBC_VER_MAX_REQUIRED"
else
    echo    "⚠ $ISSUES binaries need attention"
    echo    "    Some binaries require newer GLIBC than $GLIBC_VER_MAX_REQUIRED"
    echo    "    These binaries may not work on older Linux distributions."
    if [ -z "$IVPN_BUILD_SKIP_GLIBC_VER_CHECK" ] && [ -z "$GITHUB_ACTIONS" ]; 
    then
      read -p "[?] Continue anyway? [y/N]: " yn
      case $yn in
          [Yy]* ) 
              echo "[i] Continuing with build despite GLIBC compatibility issues..."
              ;;
          * ) 
              echo "[ ] Build stopped by user due to GLIBC compatibility concerns"
              exit 1
              ;;
      esac
    fi
fi
echo "======================================================"

set +e
