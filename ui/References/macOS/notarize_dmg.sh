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
# version info variables
_VERSION=""

# reading version info from arguments
while getopts ":v:c:f:" opt; do
  case $opt in
    v) _VERSION="$OPTARG"
    ;;
    c) _SIGN_CERT="$OPTARG"
    ;;
    f) _PATH_DMG_FILE="$OPTARG"
    ;;
  esac
done

if [ -z "${_VERSION}" ] || [ -z "${_SIGN_CERT}" ]; then
  echo "Usage:"
  echo "    $0 -v <version> -c <APPLE_DEVID_SERT>"
  exit 1
fi

_PATH_COMPILED_FOLDER=${_SCRIPT_DIR}/_compiled

if [ ! -f ${_PATH_DMG_FILE} ]; then
  echo "ERROR: Unable to notarize. File not exists '${_PATH_DMG_FILE}'"
  exit 1
fi

echo "[ ] *** Ready to send for notarization ***"
echo "    Version:                 '${_VERSION}'"
echo "    Apple DevID certificate: '${_SIGN_CERT}'"
echo "    File to notarize:        '${_PATH_DMG_FILE}'"
echo " "

_NOTARIZATION_SENT=0
echo " *** [APPLE NOTARIZATION] Do you wish to upload '${_PATH_DMG_FILE}' to Apple for notarization? *** "
read -p "(y\n)" yn
    case $yn in
        [Yy]* )
          echo "UPLOADING TO APPLE NOTARIZATION SERVICE...";
          read -p  'Apple credentials - Username (email): ' _uservar
          read -sp 'Apple credentials - Password        : ' _passvar
          echo ""
          echo "Uploading (will take few minutes of time, no progress indication) ..."
          xcrun altool --notarize-app --primary-bundle-id "${_VERSION}" \
                        --username "${_uservar}" \
                        --password "${_passvar}" \
                        --file "${_PATH_DMG_FILE}"
          CheckLastResult;
          _NOTARIZATION_SENT=1
          ;;
        [Nn]* )
          echo "Apple notarization skipped."
          ;;
        * ) ;;
    esac

    if [[ ${_NOTARIZATION_SENT} == 1 ]]; then
      echo "--------------------------------------------"
      echo " *** Do you wish to stample Apple notarization result to a file? *** "
      echo "    [NOTE!] Before doing that, you must wait until Apple service"
      echo "            will finish notarization process with 'Package Approved' result."
      echo "            Usually, it takes less than a hour."
      echo "            Untill that, you can leave this script opened (do not answer 'y')."
      echo ""
      echo "    Usefull commands (you can execute them in another terminal):  "
      echo "        To check notarization history:  "
      echo "            xcrun altool --notarization-history 0 -u <APPLE_NOTARIZATION_USER> "
      echo "        To check notarization status of concrete package:  "
      echo "            xcrun altool --notarization-info <RequestUUID> -u <APPLE_NOTARIZATION_USER> "
      read -p "(y\n)" yn
          case $yn in
              [Yy]* )
                echo "STAPLING NOTARIZATION INFO...";
                xcrun stapler staple "${_PATH_DMG_FILE}"
                CheckLastResult;
                ;;
              [Nn]* );;
              * ) ;;
          esac
    fi
