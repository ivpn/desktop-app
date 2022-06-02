#!/bin/bash
cd "$( dirname "${BASH_SOURCE[0]}" )"

# if servers.json not exists or it was updated more that 1 hour - update it
# CAREFUL! (file can be updated from Git. Therefore, would be not possible to update it from website during 60 mins )
#if [[ ! -r "etc/servers.json" || $(find "etc/servers.json" -mmin +60) ]]; then

  echo "[+] UPDATING servers.json ..."

  curl -sf "https://api.ivpn.net/v5/servers.json" > ../etc/tmp_servers.json
  if ! [ $? -eq 0 ]
  then #check result of last command
    rm ../etc/tmp_servers.json
    echo "[!] ERROR: Failed to download 'servers.json'"
    exit 1
  fi

  mv ../etc/tmp_servers.json ../etc/servers.json
  if ! [ $? -eq 0 ]
  then #check result of last command
    echo "[!] ERROR: Failed to update 'servers.json'"
    exit 1
  fi

  echo "[i] Updated: etc/servers.json"

#fi
