#!/bin/bash
cd "$( dirname "${BASH_SOURCE[0]}" )"

# if servers.json not exists or it was updated more that 1 hour - update it
# CAREFUL! (file can be updated from Git. Therefore, would be not possible to update it from website during 60 mins )
#if [[ ! -r "etc/servers.json" || $(find "etc/servers.json" -mmin +60) ]]; then

  COMMON_ETC_PATH=../../common/etc
  SERVERS_FILE=${COMMON_ETC_PATH}/servers.json

  mkdir -p ${COMMON_ETC_PATH}

  echo "[+] UPDATING servers.json ..."

  curl -sf "https://api.ivpn.net/v5/servers.json" > ${COMMON_ETC_PATH}/tmp_servers.json
  if ! [ $? -eq 0 ]
  then #check result of last command
    rm ${COMMON_ETC_PATH}/tmp_servers.json
    echo "[!] ERROR: Failed to download 'servers.json'"
    
    if [ -f ${SERVERS_FILE} ]; then
      # In case of failure to download the latest version of 'servers.json', we can use the local copy of it.
      # To enable this ability, the environment variable has to be defined: IVPN_BUILD_CAN_SKIP_DOWNLOAD_SERVERS
      # (It can be useful for situations, for example, when https://api.ivpn.net is blocked by ISP)
      if [ ! -z "$IVPN_BUILD_CAN_SKIP_DOWNLOAD_SERVERS" ]; then
          echo "[i] Using local copy of 'servers.json': ${SERVERS_FILE}"
          exit 0
      fi
    fi

    exit 1
  fi

  mv ${COMMON_ETC_PATH}/tmp_servers.json ${SERVERS_FILE}
  if ! [ $? -eq 0 ]
  then #check result of last command
    echo "[!] ERROR: Failed to update 'servers.json'"
    exit 1
  fi

  echo "[i] Updated: ${SERVERS_FILE}"

#fi
