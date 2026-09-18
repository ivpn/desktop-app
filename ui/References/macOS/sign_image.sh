#!/bin/bash

#save current dir
_BASE_DIR="$( pwd )"
_SCRIPT=`basename "$0"`
#enter the script folder
cd "$(dirname "$0")"
_SCRIPT_DIR="$( pwd )"

# check result of last executed command
function CheckLastResult
{
  if ! [ $? -eq 0 ]; then #check result of last command
    if [ -n "$1" ]; then
      echo $1
    else
      echo "FAILED"
    fi
    exit 1
  fi
}

# The Apple DevID certificate which will be used to sign binaries
_SIGN_CERT=""
# reading version info from arguments
while getopts ":c:" opt; do
  case $opt in
    c) _SIGN_CERT="$OPTARG"
    ;;
  esac
done

if [ -z "${_SIGN_CERT}" ]; then
  echo "ERROR: Apple DevID not defined"
  echo "Usage:"
  echo "    $0 -c <APPLE_DEVID_SERT> [-libivpn]"
  exit 1
fi

ARCH_TARGET="${ARCH_TARGET:-$(uname -m)}"
_IMAGE_DIR="_image/${ARCH_TARGET}"

if [ ! -d "${_IMAGE_DIR}/IVPN.app" ]; then
  echo "ERROR: folder not exists '${_IMAGE_DIR}/IVPN.app'!"
fi

echo "[i] Signing by cert: '${_SIGN_CERT}'"

# temporarily setting the IFS (internal field seperator) to the newline character.
# (required to process result pf 'find' command)
IFS=$'\n'; set -f

echo "[+] Signing obfsproxy libraries..."
for f in $(find "${_IMAGE_DIR}/IVPN.app/Contents/Resources/obfsproxy" -name '*.so');
do
  echo "    signing: [" $f "]";
  codesign --verbose=4 --force --sign "${_SIGN_CERT}" "$f"
  CheckLastResult "Signing failed"
done

#restore temporarily setting the IFS (internal field seperator)
unset IFS; set +f

ListCompiledLibs=()
if [[ "$@" == *"-libivpn"* ]]
then
  ListCompiledLibs=(
  "${_IMAGE_DIR}/IVPN.app/Contents/MacOS/libivpn.dylib"
  )
fi

ListCompiledBinaries=(
"${_IMAGE_DIR}/IVPN.app/Contents/MacOS/IVPN"
"${_IMAGE_DIR}/IVPN.app/Contents/MacOS/IVPN Agent"
"${_IMAGE_DIR}/IVPN.app/Contents/MacOS/cli/ivpn"
"${_IMAGE_DIR}/IVPN.app/Contents/MacOS/kem/kem-helper"
"${_IMAGE_DIR}/IVPN.app/Contents/MacOS/IVPN Installer.app/Contents/MacOS/IVPN Installer"
"${_IMAGE_DIR}/IVPN.app/Contents/MacOS/IVPN Installer.app"
"${_IMAGE_DIR}/IVPN.app"
"${_IMAGE_DIR}/IVPN Uninstaller.app"
"${_IMAGE_DIR}/IVPN Uninstaller.app/Contents/MacOS/IVPN Uninstaller"
)

ListThirdPartyBinaries=(
"${_IMAGE_DIR}/IVPN.app/Contents/MacOS/IVPN Installer.app/Contents/Library/LaunchServices/net.ivpn.client.Helper"
"${_IMAGE_DIR}/IVPN.app/Contents/MacOS/net.ivpn.LaunchAgent"
"${_IMAGE_DIR}/IVPN.app/Contents/MacOS/openvpn"
"${_IMAGE_DIR}/IVPN.app/Contents/MacOS/WireGuard/wg"
"${_IMAGE_DIR}/IVPN.app/Contents/MacOS/WireGuard/wireguard-go"
"${_IMAGE_DIR}/IVPN.app/Contents/Resources/obfsproxy/obfs4proxy"
"${_IMAGE_DIR}/IVPN.app/Contents/MacOS/v2ray/v2ray"
"${_IMAGE_DIR}/IVPN.app/Contents/MacOS/dnscrypt-proxy/dnscrypt-proxy"
)

echo "[+] Signing compiled libs..."
for f in "${ListCompiledLibs[@]}";
do
  echo "    signing: [" $f "]";
  codesign --verbose=4 --force --sign "${_SIGN_CERT}" "$f"
  CheckLastResult "Signing failed"
done

echo "[+] Signing third-party binaries..."
for f in "${ListThirdPartyBinaries[@]}";
do
  echo "    signing: [" $f "]";
  codesign --verbose=4 --force --sign "${_SIGN_CERT}" --options runtime "$f"
  CheckLastResult "Signing failed"
done

echo "[+] Signing compiled binaries..."
for f in "${ListCompiledBinaries[@]}";
do
  echo "    signing: [" $f "]";
  codesign --verbose=4 --force --sign "${_SIGN_CERT}" --options runtime "$f" --deep --entitlements build_HarderingEntitlements.plist
  CheckLastResult "Signing failed"
done

# IVPN.app's --deep pass above just re-signed everything nested inside it, including
# the Split Tunnel system extension - overwriting its distinct entitlements
# (NetworkExtension/App Groups only, no Hardened Runtime relaxations) with
# build_HarderingEntitlements.plist's. Re-sign it here to restore the correct ones.
_ST_EXT_BUNDLE="${_IMAGE_DIR}/IVPN.app/Contents/Library/SystemExtensions/com.electron.ivpn-ui.SplitTunnel.systemextension"
if [ -d "${_ST_EXT_BUNDLE}" ]; then
  echo "[+] Re-signing Split Tunnel system extension (own entitlements)..."
  codesign --verbose=4 --force --timestamp --sign "${_SIGN_CERT}" --options runtime \
    --entitlements "HelperProjects/SplitTunnelExtension/splittunnel.entitlements" "${_ST_EXT_BUNDLE}"
  CheckLastResult "Signing failed"

  # IVPN.app's own CodeResources sealed the extension's previous (pre-re-sign) bytes -
  # changing them above now makes `codesign --verify --deep` fail with "a sealed
  # resource is missing or invalid" (confirmed empirically). Re-sign IVPN.app once
  # more, WITHOUT --deep, so it reseals around the extension's now-final signature
  # instead of re-signing (and re-clobbering) it again.
  echo "[+] Re-signing IVPN.app (no --deep) to reseal around the extension's final signature..."
  codesign --verbose=4 --force --sign "${_SIGN_CERT}" --options runtime \
    --entitlements build_HarderingEntitlements.plist "${_IMAGE_DIR}/IVPN.app"
  CheckLastResult "Signing failed"
fi
