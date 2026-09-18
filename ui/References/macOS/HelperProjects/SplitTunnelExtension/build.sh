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

# Must match CFBundleIdentifier in Info.plist
_BUNDLE_ID="com.electron.ivpn-ui.SplitTunnel"

_VERSION=""
# Embedded provisioning profile: optional here - a dev/CI build can skip it, but a real
# System Extension activation requires it (same convention as _SIGN_CERT elsewhere).
_PROVISION_PROFILE=""

while getopts ":v:E:" opt; do
  case $opt in
    v) _VERSION="$OPTARG"
    ;;
    E) _PROVISION_PROFILE="$OPTARG"
    ;;
  esac
done

if [ -z "${_VERSION}" ]; then
  echo "Usage:"
  echo "    $0 -v <version> [-E <path to embedded.provisionprofile>]"
  exit 1
fi

# ====== Architecture setup (single arch per invocation, matching every other
# HelperProjects build.sh and the outer ui/References/macOS/build.sh - IVPN.app
# itself is never shipped as a universal binary, so this isn't lipo'd either) ======
_HOST_ARCH="$(uname -m)"
ARCH_TARGET="${ARCH_TARGET:-$_HOST_ARCH}"
case "$ARCH_TARGET" in
  arm64)  _ARCH_FLAG="-arch arm64" ;;
  x86_64) _ARCH_FLAG="-arch x86_64" ;;
  *)
    echo "ERROR: Unsupported ARCH_TARGET='$ARCH_TARGET'. Use 'arm64' or 'x86_64'."
    exit 1
    ;;
esac
_DEPLOY_MIN="12.0"
_SDK="$(xcrun --sdk macosx --show-sdk-path)"
echo "    ARCH_TARGET: ${ARCH_TARGET}"
# ====== End architecture setup ======

echo "[ ] *** Compiling IVPN Split Tunnel system extension ***"
echo "    Version: '${_VERSION}'"

_OUT_DIR="bin/${ARCH_TARGET}"
_BUNDLE="${_OUT_DIR}/${_BUNDLE_ID}.systemextension"

rm -fr "${_OUT_DIR}"
CheckLastResult

echo "[+] Updating Info.plist version info ..."
_PLIST_INFO="${_BUNDLE}/Contents/Info.plist"
mkdir -p "${_BUNDLE}/Contents" || CheckLastResult
# Stamp a copy, not the checked-in Info.plist - plutil -replace rewrites the whole
# file and would silently drop its explanatory comments.
cp "Info.plist" "${_PLIST_INFO}" || CheckLastResult
plutil -replace CFBundleShortVersionString -xml "<string>${_VERSION}</string>" "${_PLIST_INFO}" || CheckLastResult
plutil -replace CFBundleVersion -xml "<string>${_VERSION}</string>" "${_PLIST_INFO}" || CheckLastResult

echo "[+] Compiling ..."
mkdir -p "${_BUNDLE}/Contents/MacOS" || CheckLastResult
clang ${_ARCH_FLAG} -isysroot ${_SDK} -mmacosx-version-min=${_DEPLOY_MIN} -fobjc-arc \
      -framework Foundation -framework NetworkExtension -framework Network -lbsm \
      main.m STLog.m STPathMatching.m STPhysicalInterfaceSelector.m \
      STProxyProvider.m "STProxyProvider+TCPRelay.m" "STProxyProvider+UDPRelay.m" \
      -o "${_BUNDLE}/Contents/MacOS/SplitTunnelExtension"
CheckLastResult

if [ -n "${_PROVISION_PROFILE}" ]; then
  echo "[+] Embedding provisioning profile ..."
  cp "${_PROVISION_PROFILE}" "${_BUNDLE}/Contents/embedded.provisionprofile" || CheckLastResult
else
  echo "[!] WARNING: no provisioning profile supplied (-E) - the extension won't be able to activate as a real System Extension"
fi

# Signing happens later, in sign_image.sh: it needs the extension's own entitlements
# (splittunnel.entitlements, not build_HarderingEntitlements.plist) applied *after*
# the outer IVPN.app's --deep signing pass, which would otherwise overwrite them.

echo "[ ] Done. Extension bundle: '${_SCRIPT_DIR}/${_BUNDLE}'"

cd ${_BASE_DIR}
