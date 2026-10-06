#!/bin/bash

#save current dir
_BASE_DIR="$( pwd )"
_SCRIPT=`basename "$0"`
#enter the script folder
cd "$(dirname "$0")"
_SCRIPT_DIR="$( pwd )"

_BUILDTAGS_USE_LIBVPN="" # can be a '-libivpn' to  use XPC listener for notifying clients about daemon connection port (latest IVPN UI not using XPC)

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

# The Apple DevID certificate which will be used to sign binaries (example: WQXXXXXBYN)
_SIGN_CERT=""
# version info variables
_VERSION=""

_FILE_TO_INTEGRATE_IN_BUNDLE=""
# Embedded provisioning profiles for the Split Tunnel feature (System Extension +
# App Groups entitlements need one on both the host app and the extension). Optional:
# a build without them just can't activate a real System Extension (today's status quo).
_HOST_PROVISION_PROFILE=""
_EXT_PROVISION_PROFILE=""
# reading version info from arguments
while getopts ":v:c:i:P:E:" opt; do
  case $opt in
    v) _VERSION="$OPTARG"
    ;;
    c) _SIGN_CERT="$OPTARG"
    ;;
    i) _FILE_TO_INTEGRATE_IN_BUNDLE="$OPTARG"
    ;;
    P) _HOST_PROVISION_PROFILE="$OPTARG"
    ;;
    E) _EXT_PROVISION_PROFILE="$OPTARG"
    ;;
  esac
done

# The two profiles only work as a pair: a host app without its profile cannot
# activate the extension, and an extension without its profile fails to load.
if { [ -n "${_HOST_PROVISION_PROFILE}" ] && [ -z "${_EXT_PROVISION_PROFILE}" ]; } || \
   { [ -z "${_HOST_PROVISION_PROFILE}" ] && [ -n "${_EXT_PROVISION_PROFILE}" ]; }; then
  echo "[!] ERROR: -P and -E must be given together (host app and Split Tunnel extension provisioning profiles)."
  exit 1
fi
for _profile in "${_HOST_PROVISION_PROFILE}" "${_EXT_PROVISION_PROFILE}"; do
  if [ -n "${_profile}" ] && [ ! -f "${_profile}" ]; then
    echo "[!] ERROR: provisioning profile not found: ${_profile}"
    exit 1
  fi
done

if [ -z "${_VERSION}" ]; then
  _VERSION="$(awk -F'"' '/"version"/{print $4; exit}' "${_SCRIPT_DIR}/../../package.json" 2>/dev/null)"
  if [ -z "${_VERSION}" ]; then
    echo "[!] ERROR: Could not detect version from package.json. Use -v <version>."
    exit 1
  fi
  echo "[i] Version auto-detected: ${_VERSION}"
fi

echo "[+] *** COMPILING IVPN BINARIES AND MAKING DMG ***";
echo "    Version:                 '${_VERSION}'"
if [ -z "${_SIGN_CERT}" ]; then
  if [ ! -z "$GITHUB_ACTIONS" ]; then
    echo "! GITHUB_ACTIONS detected ! It is just a build test."
    echo "! No signinnng certificate required !"
  else
    echo "    ERROR: Apple DevID not defined (signing & notarization will be skipped)"
    echo "           It is not possible to build helper and uninstaller projects."
    echo "           Signing & notarization not possible too."
    echo "    Usage:"
    echo "          $0 -v <version> -c <APPLE_DEVID_CERTIFICATE>"
    exit 1
  fi
else
  echo "    Apple DevID certificate: '${_SIGN_CERT}'"
fi

_PATH_REL_REPO_DAEMON="./../../../daemon"
_PATH_REL_REPO_CLI="./../../../cli"
_PATH_REL_REPO_UI="./../.."
_PATH_ABS_REPO_DAEMON=""
_PATH_ABS_REPO_CLI=""
_PATH_ABS_REPO_UI=""

# ============================== ARCHITECTURE =============================
_HOST_ARCH="$(uname -m)"
ARCH_TARGET="${ARCH_TARGET:-$_HOST_ARCH}"
case "$ARCH_TARGET" in
  arm64)  _COMPILEDFOLDER="mac-arm64" ;;
  x86_64) _COMPILEDFOLDER="mac" ;;
  *)
    echo "ERROR: Unsupported ARCH_TARGET='$ARCH_TARGET'. Use 'arm64' or 'x86_64'."
    exit 1
    ;;
esac
export ARCH_TARGET
echo "    ARCH_TARGET:             '${ARCH_TARGET}'"

# ============================== PROJECTS PATH =============================
# obtaining absolute paths to sources of daemon and CLI
if [ ! -d ${_PATH_REL_REPO_DAEMON} ]; then
  echo "[!] ERROR: daemon project not exists: '${_PATH_REL_REPO_DAEMON}'"
  exit -1
fi
if [ ! -d ${_PATH_REL_REPO_CLI} ]; then
  echo "[!] ERROR: daemon project not exists: '${_PATH_REL_REPO_CLI}'"
  exit -1
fi
if [ ! -d ${_PATH_REL_REPO_UI} ]; then
  echo "[!] ERROR: UI sources folder not exists: '${_PATH_REL_REPO_UI}'"
  exit -1
fi
cd ${_PATH_REL_REPO_DAEMON} || CheckLastResult
_PATH_ABS_REPO_DAEMON="$( pwd )"
CheckLastResult "[!] ERROR obtaining absolute path to daemon project"

cd ${_SCRIPT_DIR}
cd ${_PATH_REL_REPO_CLI} || CheckLastResult
_PATH_ABS_REPO_CLI="$( pwd )"
CheckLastResult "[!] ERROR obtaining absolute path to CLI project"

cd ${_SCRIPT_DIR}
cd ${_PATH_REL_REPO_UI} || CheckLastResult
_PATH_ABS_REPO_UI="$( pwd )"
CheckLastResult "[!] ERROR obtaining absolute path to CLI project"

cd ${_SCRIPT_DIR}

echo "    UI sources:     ${_PATH_ABS_REPO_UI}"
echo "    Daemon sources: ${_PATH_ABS_REPO_DAEMON}"
echo "    CLI sources:    ${_PATH_ABS_REPO_CLI}"

# ============================== BUILDING PROJECTS =============================
echo "[+] Building IVPN Daemon (${_PATH_ABS_REPO_DAEMON})...";
ARCH_TARGET="${ARCH_TARGET}" ${_PATH_ABS_REPO_DAEMON}/References/macOS/scripts/build-all.sh -norebuild -wifi ${_BUILDTAGS_USE_LIBVPN} -v ${_VERSION}
CheckLastResult "[!] ERROR building IVPN Daemon"

echo "[+] Building helper ..."
if [ -z "${_SIGN_CERT}" ]; then
  ARCH_TARGET="${ARCH_TARGET}" ${_PATH_ABS_REPO_UI}/References/macOS/HelperProjects/helper/build.sh -v ${_VERSION}
else
  ARCH_TARGET="${ARCH_TARGET}" ${_PATH_ABS_REPO_UI}/References/macOS/HelperProjects/helper/build.sh -v ${_VERSION} -c ${_SIGN_CERT}
fi
CheckLastResult "[!] ERROR building helper binary"

echo "[+] Building LaunchAgent (net.ivpn.LaunchAgent) ..."
ARCH_TARGET="${ARCH_TARGET}" ${_PATH_ABS_REPO_UI}/References/macOS/HelperProjects/launchAgent/build.sh
CheckLastResult "[!] ERROR building net.ivpn.LaunchAgent binary"

if [ ! -z ${_BUILDTAGS_USE_LIBVPN} ]; then
  echo "[+] Building libivpn.dylib ..."
  cd ${_PATH_ABS_REPO_UI}/References/macOS/HelperProjects/libivpn
  make
  CheckLastResult "[!] ERROR building libivpn.dylib"
fi

cd ${_SCRIPT_DIR}

echo "[+] Building Uninstaller/Installer ..."
ARCH_TARGET="${ARCH_TARGET}" ${_PATH_ABS_REPO_UI}/References/macOS/HelperProjects/uninstaller/build.sh -c ${_SIGN_CERT}
CheckLastResult "[!] ERROR building Uninstaller/Installer"
cd ${_SCRIPT_DIR}

echo "[+] Building Split Tunnel system extension ..."
_ST_EXT_BUILD_ARGS=()
[ -n "${_EXT_PROVISION_PROFILE}" ] && _ST_EXT_BUILD_ARGS=(-E "${_EXT_PROVISION_PROFILE}")
ARCH_TARGET="${ARCH_TARGET}" ${_PATH_ABS_REPO_UI}/References/macOS/HelperProjects/SplitTunnelExtension/build.sh -v ${_VERSION} "${_ST_EXT_BUILD_ARGS[@]}"
CheckLastResult "[!] ERROR building Split Tunnel system extension"
cd ${_SCRIPT_DIR}

echo "[+] Building IVPN CLI (${_PATH_ABS_REPO_CLI})...";
ARCH_TARGET="${ARCH_TARGET}" ${_PATH_ABS_REPO_CLI}/References/macOS/build.sh -v ${_VERSION}
CheckLastResult "[!] ERROR building IVPN CLI"

echo ======================================================
echo ================= Compiling UI =======================
echo ======================================================

echo "[+] Building UI (${_PATH_ABS_REPO_UI})...";
cd ${_PATH_ABS_REPO_UI}
echo "[+] Building UI: Installing NPM molules ..."
npm install
CheckLastResult
echo "[+] Building UI: Build..."
if [ "${ARCH_TARGET}" = "arm64" ]; then
    npm run electron:build:mac:arm64
else
    npm run electron:build:mac:x64
fi
CheckLastResult

# ============================== STOP IF GITHUB_ACTIONS ==============================
if [ ! -z "$GITHUB_ACTIONS" ]; then
  echo "! GITHUB_ACTIONS detected !"
  echo "(skipped: signing; DMG creation)"
  echo ""
  echo "GITHUB_ACTIONS Build TEST: SUCCESS"
  exit 0
fi

# ============================== PREPARING DMG ==============================
echo ======================================================
echo ================= Preparing DMG ======================
echo ======================================================
echo "[ ] Architecture: ${ARCH_TARGET} | Electron folder: ${_COMPILEDFOLDER}"

# Copy a binary to destination and verify it matches ARCH_TARGET.
# Fails immediately if the file is missing or has the wrong arch.
CopyBinary() {
    local src="$1"
    local dst="$2"
    if [ ! -f "${src}" ]; then
        echo "[!] ERROR: missing binary: ${src}"; exit 1
    fi
    local arch
    arch=$(lipo -info "${src}" 2>/dev/null | grep -oE 'arm64|x86_64' | tail -1)
    if [ "${arch}" != "${ARCH_TARGET}" ]; then
        echo "[!] ERROR: wrong arch (expected ${ARCH_TARGET}, got '${arch}'): $(basename "${src}")"; exit 1
    fi
    cp "${src}" "${dst}" || exit 1
    echo "    [+] ${arch}: $(basename "${src}")"
}

echo "[+] Preparing image folder..."
_PATH_IMAGE_FOLDER="${_SCRIPT_DIR}/_image/${ARCH_TARGET}"
_PATH_UI_COMPILED_IMAGE="${_PATH_IMAGE_FOLDER}/IVPN.app"
_PATH_COMPILED_UI_ORIG="${_PATH_ABS_REPO_UI}/dist/${_COMPILEDFOLDER}/IVPN.app"

rm -fr "${_PATH_IMAGE_FOLDER}"
sleep 2
mkdir -p "${_PATH_IMAGE_FOLDER}" || CheckLastResult

if [ ! -d "${_PATH_COMPILED_UI_ORIG}" ]; then
  echo "[!] ERROR: compiled UI not found: ${_PATH_COMPILED_UI_ORIG}"; exit 1
fi

# Path shorthand aliases (set after image folder is known)
_D="${_PATH_UI_COMPILED_IMAGE}/Contents/MacOS"
_R="${_PATH_UI_COMPILED_IMAGE}/Contents/Resources"
_DEPS="${_PATH_ABS_REPO_DAEMON}/References/macOS/_deps/${ARCH_TARGET}"
_HELPERS="${_PATH_ABS_REPO_UI}/References/macOS/HelperProjects"

echo "[+] Copying background and Electron app..."
mkdir -p "${_PATH_IMAGE_FOLDER}/.background"
cp "${_PATH_ABS_REPO_UI}/References/macOS/resources/dmg_background.png" \
   "${_PATH_IMAGE_FOLDER}/.background/back.png" || CheckLastResult
cp -a "${_PATH_COMPILED_UI_ORIG}" "${_PATH_UI_COMPILED_IMAGE}" || CheckLastResult
rm -f ${_PATH_ABS_REPO_UI}/dist/IVPN* # remove electron-builder's auto-generated DMGs

echo "[+] Copying 'etc' config files..."
cp -R "${_PATH_ABS_REPO_DAEMON}/References/macOS/etc"  "${_R}" || CheckLastResult
cp -R "${_PATH_ABS_REPO_DAEMON}/References/common/etc" "${_R}" || CheckLastResult
echo "    (sudo may be required for permissions)"
sudo chmod 0400 "${_R}"/etc/*.* || CheckLastResult
sudo chmod 0700 "${_R}"/etc/*.sh || CheckLastResult

echo "[+] Copying binaries (arch: ${ARCH_TARGET})..."
mkdir -p "${_D}/cli" "${_D}/WireGuard" "${_D}/v2ray" "${_D}/dnscrypt-proxy" \
         "${_D}/kem" "${_R}/obfsproxy"

CopyBinary "${_PATH_ABS_REPO_DAEMON}/bin/${ARCH_TARGET}/IVPN Agent"               "${_D}/IVPN Agent"
CopyBinary "${_PATH_ABS_REPO_CLI}/References/macOS/_out_bin/${ARCH_TARGET}/ivpn"  "${_D}/cli/ivpn"
CopyBinary "${_DEPS}/openvpn_inst/bin/openvpn"                                    "${_D}/openvpn"
CopyBinary "${_DEPS}/obfs4proxy_inst/obfs4proxy"                                  "${_R}/obfsproxy/obfs4proxy"
CopyBinary "${_DEPS}/v2ray_inst/v2ray"                                            "${_D}/v2ray/v2ray"
CopyBinary "${_DEPS}/wg_inst/wg"                                                  "${_D}/WireGuard/wg"
CopyBinary "${_DEPS}/wg_inst/wireguard-go"                                        "${_D}/WireGuard/wireguard-go"
CopyBinary "${_DEPS}/dnscryptproxy_inst/dnscrypt-proxy"                           "${_D}/dnscrypt-proxy/dnscrypt-proxy"
CopyBinary "${_DEPS}/kem-helper/kem-helper-bin/kem-helper"                        "${_D}/kem/kem-helper"
CopyBinary "${_HELPERS}/launchAgent/_out/${ARCH_TARGET}/net.ivpn.LaunchAgent"     "${_D}/net.ivpn.LaunchAgent"

echo "[+] Copying installer/uninstaller app bundles..."
# net.ivpn.client.Helper is already embedded inside IVPN Installer.app by uninstaller/build.sh
cp -R "${_HELPERS}/uninstaller/bin/${ARCH_TARGET}/IVPN Installer.app"   "${_D}"                   || CheckLastResult
cp -R "${_HELPERS}/uninstaller/bin/${ARCH_TARGET}/IVPN Uninstaller.app" "${_PATH_IMAGE_FOLDER}"   || CheckLastResult

echo "[+] Copying Split Tunnel system extension..."
_ST_EXT_BUNDLE_ID="$(/usr/libexec/PlistBuddy -c 'Print CFBundleIdentifier' "${_HELPERS}/SplitTunnelExtension/Info.plist")"
mkdir -p "${_PATH_UI_COMPILED_IMAGE}/Contents/Library/SystemExtensions"
cp -R "${_HELPERS}/SplitTunnelExtension/bin/${ARCH_TARGET}/${_ST_EXT_BUNDLE_ID}.systemextension" \
      "${_PATH_UI_COMPILED_IMAGE}/Contents/Library/SystemExtensions/" || CheckLastResult
if [ -n "${_HOST_PROVISION_PROFILE}" ]; then
  cp "${_HOST_PROVISION_PROFILE}" "${_PATH_UI_COMPILED_IMAGE}/Contents/embedded.provisionprofile" || CheckLastResult
fi

echo "[+] Copying LaunchAgent plist..."
mkdir -p "${_PATH_UI_COMPILED_IMAGE}/Contents/Library/LaunchAgents"
cp -R "${_HELPERS}/launchAgent/net.ivpn.LaunchAgent_launchd.plist" \
      "${_PATH_UI_COMPILED_IMAGE}/Contents/Library/LaunchAgents/" || CheckLastResult

if [ ! -z "${_BUILDTAGS_USE_LIBVPN}" ]; then
  echo "[+] Copying libivpn.dylib..."
  cp "${_HELPERS}/libivpn/libivpn.dylib" "${_D}" || CheckLastResult
fi

if [ ! -z "${_FILE_TO_INTEGRATE_IN_BUNDLE}" ]; then
  echo "[+] Integrating external file: ${_FILE_TO_INTEGRATE_IN_BUNDLE}..."
  cp "${_FILE_TO_INTEGRATE_IN_BUNDLE}" "${_R}" || CheckLastResult
fi

# ============================== SIGNING ==============================
if [ -z "${_SIGN_CERT}" ]; then
  echo "[!] WARNING! SIGNING CERTIFICATE NOT DEFINED"
  echo "             Signing skipped!"
else
  ${_SCRIPT_DIR}/sign_image.sh -c ${_SIGN_CERT} ${_BUILDTAGS_USE_LIBVPN}
  CheckLastResult "ERROR: SIGNING FAILED!"
fi
# ============================== GENERATING DMG ==============================
echo "[+] GENERATING DMG ..."

_PATH_COMPILED_FOLDER=${_SCRIPT_DIR}/_compiled

_PATH_DMG_FILE="${_PATH_COMPILED_FOLDER}/IVPN-"${_VERSION}".dmg"
if [ "${ARCH_TARGET}" != "x86_64" ]; then
  _PATH_DMG_FILE="${_PATH_COMPILED_FOLDER}/IVPN-"${_VERSION}-${ARCH_TARGET}".dmg"
fi

_PATH_TMP_DMG_FILE="${_PATH_COMPILED_FOLDER}/ivpn-${ARCH_TARGET}.temp.dmg"

_BACKGROUND_FILE="back.png"
_APPLICATION_NAME="IVPN.app"
_UNINSTALL_APPLICATION_NAME="IVPN Uninstaller.app"
_source=${_PATH_IMAGE_FOLDER}
_title="IVPN-${_VERSION}"
_size=409600 # max disk size (KB)

# creating output directory (if not exists)
mkdir -p ${_PATH_COMPILED_FOLDER} || CheckLastResult "Failed to create '${_PATH_COMPILED_FOLDER}'"
# Removing old files
rm -f ${_PATH_TMP_DMG_FILE}
rm -f ${_PATH_DMG_FILE}

echo "[+] Generating DMG: Creating a new temporary r/w DMG image ..."
hdiutil create -srcfolder "${_source}" -volname "${_title}" -fs HFS+ \
      -fsargs "-c c=64,a=16,e=16" -format UDRW -size ${_size}k ${_PATH_TMP_DMG_FILE}
CheckLastResult

echo "[+] Generating DMG: Mounting new temporary r/w DMG image ..."
_device=$(hdiutil attach -readwrite -noverify -noautoopen ${_PATH_TMP_DMG_FILE} | \
         egrep '^/dev/' | sed 1q | awk '{print $1}')
CheckLastResult
echo "[i] Generating DMG: Mounted as device: ${_device}"

sleep 2 # give time to finish mounting

echo "[+] Generating DMG: Updating temporary r/w DMG image ..."
echo '
   tell application "Finder"
     tell disk "'${_title}'"
           open
           set current view of container window to icon view
           set toolbar visible of container window to false
           set statusbar visible of container window to false
           set the bounds of container window to {200, 200, 758, 680}
           set theViewOptions to the icon view options of container window
           set arrangement of theViewOptions to not arranged
           set icon size of theViewOptions to 108
           set background picture of theViewOptions to file ".background:'${_BACKGROUND_FILE}'"
           make new alias file at container window to POSIX file "/Applications" with properties {name:"Applications"}
           set position of item "'${_APPLICATION_NAME}'" of container window to {120, 110}
           set position of item "'${_UNINSTALL_APPLICATION_NAME}'" of container window to {420, 300}
           set position of item "Applications" of container window to {420, 110}
           set position of item ".background" of container window to {120, 500}
           set position of item ".fseventsd" of container window to {420, 500}
           update without registering applications
           delay 3
           close
     end tell
   end tell
' | osascript
CheckLastResult

sleep 5

echo "[+] Generating DMG: chmod DMG image ..."
chmod -Rf go-w "/Volumes/${_title}"
sync
sync

echo "[+] Generating DMG: Detaching temporary DMG from ${_device} ..."
hdiutil detach ${_device}
CheckLastResult

echo "[+] Generating DMG: Converting DMG ..."
hdiutil convert ${_PATH_TMP_DMG_FILE} -format UDZO -imagekey zlib-level=9 -o "${_PATH_DMG_FILE}"
CheckLastResult
echo "[+] Generating DMG: Removing temporary DMG DMG ..."
rm -f ${_PATH_TMP_DMG_FILE}

# ============================== NOTARIZATION ==============================
if [ -z "${_SIGN_CERT}" ]; then
  echo "[!] WARNING! SIGNING CERTIFICATE NOT DEFINED"
  echo "             Notarization skipped!"
else
  ${_SCRIPT_DIR}/notarize_dmg.sh -c ${_SIGN_CERT} -v ${_VERSION} -f "${_PATH_DMG_FILE}"
  CheckLastResult "ERROR: NOTARIZATION FAILED!"
fi

echo "[+] SCCESS"

open ${_PATH_COMPILED_FOLDER}
# restore default folder
cd ${_BASE_DIR}
