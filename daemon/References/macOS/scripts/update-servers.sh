#!/bin/bash
cd "$( dirname "${BASH_SOURCE[0]}" )"

# if servers.json not exists or it was updated more that 1 hour - update it
# CAREFUL! (file can be updated from Git. Therefore, would be not possible to update it from website during 60 mins )
#if [[ ! -r "etc/servers.json" || $(find "etc/servers.json" -mmin +60) ]]; then

  COMMON_ETC_PATH=../../common/etc
  mkdir -p ${COMMON_ETC_PATH}

  echo "======================================================"
  echo "============== UPDATING servers.json ================="
  echo "======================================================"

  curl -sf "https://api.ivpn.net/v5/servers.json" > ${COMMON_ETC_PATH}/tmp_servers.json
  if ! [ $? -eq 0 ]
  then #check result of last command
    rm ${COMMON_ETC_PATH}/tmp_servers.json
    echo "ERROR: Failed to download 'servers.json'"
    echo "======================================================"
    exit 1
  fi

  mv ${COMMON_ETC_PATH}/tmp_servers.json ${COMMON_ETC_PATH}/servers.json
  if ! [ $? -eq 0 ]
  then #check result of last command
    echo "ERROR: Failed to update 'servers.json'"
    echo "======================================================"
    exit 1
  fi

  echo "Updated: ${COMMON_ETC_PATH}/servers.json"

#fi
