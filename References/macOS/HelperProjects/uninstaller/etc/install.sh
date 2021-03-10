#!/bin/sh

_source_dmg=$1
_signature_file=$2
_signature_file_tmp_decoded="$2.decoded"
_pub_key_file="/Applications/IVPN.app/Contents/Resources/public.pem"

_volume=""

_app_path="/Applications/IVPN.app"
_app_plist="${_app_path}/Contents/Info.plist"
_app_backup="${_app_path}.old"

echoerr() { echo "[!] ERROR: $@" 1>&2; }

function UnmountDMG
{
    if [ ! -z "${_volume}" ] && [ -d "${_volume}" ]; then
        echo "[+] Unmounting '${_volume}' ..."
        hdiutil detach -quiet ${_volume}
    fi
}

function RemoveBackup
{
    if [ -d "${_app_backup}" ]; then
        echo "[+] Removing old application backup '${_app_backup}' ..."
        rm -fr "${_app_backup}" || echoerr "Failed to remove the backup: $?"
    fi
}

function RestoreBackup
{
    if [ -d "${_app_backup}" ]; then
        echo "[+] Restoring application backup '${_app_backup}' -> '${_app_path}' ..."
        mv -f "${_app_backup}" "${_app_path}" || echoerr "Failed to restore the backup"
    fi
}

function CntAppRunningProcesses
{
    return `ps aux | grep -v grep | grep -c "/Applications/IVPN.app/Contents/MacOS/IVPN"`
} 

function KillAppProcess
{
    CntAppRunningProcesses
    _cnt=$?
    if [ "${_cnt}" != "0" ]; then
        echo "Killing application process ..."
        killall "IVPN"
        sleep 1

        CntAppRunningProcesses
        _cnt=$?
        if [ "${_cnt}" != "0" ]; then
            killall "IVPN"
        fi
    fi
}

function CheckSignature
{
    echo "[+] Checking signature ..."
    /usr/bin/openssl base64 -d -in "${_signature_file}" -out "${_signature_file_tmp_decoded}" || return 1
    /usr/bin/openssl dgst -sha256 -verify "${_pub_key_file}" -signature "${_signature_file_tmp_decoded}" "${_source_dmg}" || { rm "${_signature_file_tmp_decoded}"; return 2; }
    rm "${_signature_file_tmp_decoded}"
    return 0
}

if [ -z "${_source_dmg}" ]; then
  echoerr "Source dmg file not defined."
  exit 64
fi

if [ -z "${_signature_file}" ]; then
  echoerr "Signature file not defined."
  exit 64
fi

if [ ! -f "${_pub_key_file}" ]; then
  echoerr "Public key file not exists '${_pub_key_file}'"
  exit 65
fi

if [ ! -f "${_source_dmg}" ]; then
  echoerr "Source DMG file not exists '${_source_dmg}'"
  exit 65
fi

if [ ! -f "${_signature_file}" ]; then
  echoerr "Signature file not exists '${_signature_file}'"
  exit 65
fi

CheckSignature || { echoerr "Signature check failed"; exit 60; }

echo "[+] Mounting '${_source_dmg}' ..."
_volume=`hdiutil attach -nobrowse "${_source_dmg}" | grep Volumes | awk '{print $3}'` 
if [ -z "${_volume}" ]; then
    echoerr "Failed to mount: '${_source_dmg}'"
    exit 66
fi

_app_path_src="${_volume}/IVPN.app"
_app_plist_src="${_app_path_src}/Contents/Info.plist"

if [ ! -d "${_app_path_src}" ]; then
    echoerr "Source application file not exists: '${_app_path_src}'"
    UnmountDMG
    exit 67
fi 

if [ -d "${_app_path}" ]; then

    echo "[+] Checking versions ..."
    _app_version=`defaults read "${_app_plist}" CFBundleShortVersionString`
    _app_version_src=`defaults read "${_app_plist_src}" CFBundleShortVersionString`

    if [ -z "${_app_version_src}" ]; then
        echoerr "Unable to determine update version"
        UnmountDMG
        exit 68
    fi

    if [ -z "${_app_version}" ]; then
        echoerr "Unable to determine installed version"
        UnmountDMG
        exit 69
    fi

    if [ "${_app_version}" == "${_app_version_src}" ]; then
        echoerr "Nothing to update (the version of the update is the same as currently installed)"
        UnmountDMG
        exit 70
    fi

    RemoveBackup
    echo "[+] Backup old application '${_app_path}' -> '${_app_backup}'"
    mv -f "${_app_path}" "${_app_backup}" || { echoerr "Failed to make a backup"; UnmountDMG; exit 71; }
fi

echo "[+] Copying ..."
cp -R "${_app_path_src}" "${_app_path}" || { echoerr "Failed to install the update"; RestoreBackup; UnmountDMG; exit 72; }

RemoveBackup
UnmountDMG

# Normally, the app have to be stopped on the current moment (app have to be stopped before calling the script)
# Here we just ensuring that app really closed. If no - forcing it to close
KillAppProcess

echo "[+] Starting '${_app_path}' ..."
sudo -u "$USER" open "${_app_path}"

exit 0
