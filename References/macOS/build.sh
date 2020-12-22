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

# The Apple DevID certificate which will be used to sign binaries (example: WQXXXXXBYN)
_SIGN_CERT=""
# version info variables
_VERSION=""
# reading version info from arguments
while getopts ":v:c:" opt; do
  case $opt in
    v) _VERSION="$OPTARG"
    ;;
    c) _SIGN_CERT="$OPTARG"
    ;;
  esac
done

if [ -z "${_VERSION}" ]; then
  echo "Usage:"
  echo "    $0 -v <version> -c <APPLE_DEVID_SERT>"
  exit 1
fi

echo "[+] *** COMPILING IVPN BINARIES AND MAKING DMG ***";
echo "    Version:                 '${_VERSION}'"
if [ -z "${_SIGN_CERT}" ]; then
  echo "    WARNING: Apple DevID not defined (skipping signing binaries)"
  read -p "Press enter to continue"
else
  echo "    Apple DevID certificate: '${_SIGN_CERT}'"
fi

_PATH_REL_REPO_DAEMON="./../../../desktop-app-daemon"
_PATH_REL_REPO_CLI="./../../../desktop-app-cli"
_PATH_REL_REPO_UI="./../.."
_PATH_ABS_REPO_DAEMON=""
_PATH_ABS_REPO_CLI=""
_PATH_ABS_REPO_UI=""

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

echo "[+] Checking UI project version..."
cat "${_PATH_ABS_REPO_UI}/package.json" | grep \"version\" | grep \"${_VERSION}\"
CheckLastResult "ERROR: Please set correct version in file '${_PATH_ABS_REPO_UI}/package.json'"

# ============================== BUILDING PROJECTS =============================
echo "[+] Building IVPN Daemon (${_PATH_ABS_REPO_DAEMON})...";
#${_PATH_ABS_REPO_DAEMON}/References/macOS/scripts/build-all.sh -norebuild -v ${_VERSION}
CheckLastResult "[!] ERROR building IVPN Daemon"

echo "[+] Building helper ..."
if [ -z "${_SIGN_CERT}" ]; then
  ${_PATH_ABS_REPO_UI}/References/macOS/HelperProjects/helper/build.sh -v ${_VERSION} -c ${_SIGN_CERT}
else
  ${_PATH_ABS_REPO_UI}/References/macOS/HelperProjects/helper/build.sh -v ${_VERSION}
fi
CheckLastResult "[!] ERROR building helper binary"

echo "[+] Building libivpn.dylib ..."
cd ${_PATH_ABS_REPO_UI}/References/macOS/HelperProjects/libivpn
make
CheckLastResult "[!] ERROR building libivpn.dylib"
cd ${_SCRIPT_DIR}

echo "[+] Building IVPN CLI (${_PATH_ABS_REPO_CLI})...";
#${_PATH_ABS_REPO_CLI}/References/macOS/build.sh -v ${_VERSION}
CheckLastResult "[!] ERROR building IVPN CLI"

echo "[+] Building UI (${_PATH_ABS_REPO_UI})...";
cd ${_PATH_ABS_REPO_UI}
echo "[+] Building UI: Installing NPM molules ..."
#npm install
CheckLastResult
echo "[+] Building UI: Build..."
#npm run electron:build
CheckLastResult

# ============================== PREPARING DMG ==============================
echo "[+] Preparing DMG ..."
_FNAME_UI_COMPILED="IVPN.app"
_PATH_IMAGE_FOLDER="${_SCRIPT_DIR}/_image"
_PATH_UI_COMPILED_IMAGE=${_PATH_IMAGE_FOLDER}/${_FNAME_UI_COMPILED}

_FNAME_UI_ORIG="ivpn-ui.app"
_PATH_COMPILED_UI_ORIG="${_PATH_ABS_REPO_UI}/dist_electron/mac/${_FNAME_UI_ORIG}"

# Erasing old files
rm -rf ${_PATH_IMAGE_FOLDER}
mkdir ${_PATH_IMAGE_FOLDER}  || CheckLastResult

if [ ! -d ${_PATH_COMPILED_UI_ORIG} ]; then
  echo "[!] ERROR: unable to find compiled UI binary: ${_PATH_COMPILED_UI_ORIG}"
fi

echo "[+] Preparing DMG image: Copying UI binaries ..."
cp -a "${_PATH_COMPILED_UI_ORIG}" ${_PATH_UI_COMPILED_IMAGE} || CheckLastResult

echo "[+] Preparing DMG image: Copying 'etc' ..."
cp -R "${_PATH_ABS_REPO_DAEMON}/References/macOS/etc" "${_PATH_UI_COMPILED_IMAGE}/Contents/Resources" || CheckLastResult
echo "[+] Preparing DMG image: Setting correct file permissions for 'etc' folder ..."
echo "    (sudo pass can be asked now)"
sudo chmod 0400 ${_PATH_UI_COMPILED_IMAGE}/Contents/Resources/etc/*.* || CheckLastResult
sudo chmod 0700 ${_PATH_UI_COMPILED_IMAGE}/Contents/Resources/etc/*.sh || CheckLastResult

echo "[+] Preparing DMG image: Copying 'openvpn'..."
cp "${_PATH_ABS_REPO_DAEMON}/References/macOS/_deps/openvpn_inst/bin/openvpn" "${_PATH_UI_COMPILED_IMAGE}/Contents/MacOS/openvpn" || CheckLastResult

echo "[+] Preparing DMG image: Copying 'obfsproxy' binaries..."
cp -R "${_PATH_ABS_REPO_DAEMON}/References/macOS/obfsproxy" "${_PATH_UI_COMPILED_IMAGE}/Contents/Resources" || CheckLastResult

echo "[+] Preparing DMG image: Copying 'WireGuard' binaries..."
mkdir -p "${_PATH_UI_COMPILED_IMAGE}/Contents/MacOS/WireGuard"
cp "${_PATH_ABS_REPO_DAEMON}/References/macOS/_deps/wg_inst/wg" "${_PATH_UI_COMPILED_IMAGE}/Contents/MacOS/WireGuard/wg" || CheckLastResult
cp "${_PATH_ABS_REPO_DAEMON}/References/macOS/_deps/wg_inst/wireguard-go" "${_PATH_UI_COMPILED_IMAGE}/Contents/MacOS/WireGuard/wireguard-go" || CheckLastResult

echo "[+] Preparing DMG image: Copying daemon..."
cp -R "${_PATH_ABS_REPO_DAEMON}/IVPN Agent" "${_PATH_UI_COMPILED_IMAGE}/Contents/MacOS" || CheckLastResult

echo "[+] Preparing DMG image: Copying CLI..."
mkdir "${_PATH_UI_COMPILED_IMAGE}/Contents/MacOS/cli" || CheckLastResult
cp -R "${_PATH_ABS_REPO_CLI}/References/macOS/_out_bin/ivpn" "${_PATH_UI_COMPILED_IMAGE}/Contents/MacOS/cli" || CheckLastResult

echo "[+] Preparing DMG image: Copying net.ivpn.client.Helper ..."
mkdir -p "${_PATH_UI_COMPILED_IMAGE}/Contents/Library/LaunchServices" || CheckLastResult
cp "${_PATH_ABS_REPO_UI}/References/macOS/HelperProjects/helper/net.ivpn.client.Helper" "${_PATH_UI_COMPILED_IMAGE}/Contents/Library/LaunchServices"
CheckLastResult

echo "[+] Preparing DMG image: Copying libivpn.dylib ..."
cp "${_PATH_ABS_REPO_UI}/References/macOS/HelperProjects/libivpn/libivpn.dylib" "${_PATH_UI_COMPILED_IMAGE}/Contents/MacOS"
CheckLastResult

echo "[+] Preparing DMG image: Copying background image for DMG ..."
mkdir -p "${_PATH_IMAGE_FOLDER}/.background" && cp "${_PATH_ABS_REPO_UI}/References/macOS/resources/dmg_background.png" "${_PATH_IMAGE_FOLDER}/.background/back.png"
CheckLastResult

#echo "Copying Uninstaller ..."
#cp -a "${UNINSTALL_FILE}" ./_image/
#CheckLastResult "Error copying ${UNINSTALL_FILE}"

echo "[+] Preparing DMG image: Removing unnecessary debug files..."
find "${_PATH_UI_COMPILED_IMAGE}/Contents/Resources/obfsproxy" -iname "*.pyc" -type f -delete || CheckLastResult

#echo "[+] Preparing DMG image: Signing..."
#../sign-file.sh "./_image/IVPN.app" || CheckLastResult

# ============================== GENERATING DMG ==============================
echo "[+] GENERATING DMG ..."
_PATH_COMPILED_FOLDER=${_SCRIPT_DIR}/_compiled
_PATH_DMG_FILE="${_PATH_COMPILED_FOLDER}/IVPN-"${_VERSION}".dmg"
_PATH_TMP_DMG_FILE="${_PATH_COMPILED_FOLDER}/ivpn.temp.dmg"

_BACKGROUND_FILE="back.png"
_APPLICATION_NAME="IVPN.app"
_source=${_PATH_IMAGE_FOLDER}
_title=IVPN
_size=256000

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
chmod -Rf go-w /Volumes/"${_title}"
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

echo "[+] SCCESS"

open ${_PATH_COMPILED_FOLDER}

# restore default folder
cd ${_BASE_DIR}
