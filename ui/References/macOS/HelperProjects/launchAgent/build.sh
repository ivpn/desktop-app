#!/bin/bash

#save current dir
_BASE_DIR="$( pwd )"
_SCRIPT=`basename "$0"`
#enter the script folder
cd "$(dirname "$0")"

_OUT_FOLDER="_out"
_OUT_BINARY="${_OUT_FOLDER}/net.ivpn.LaunchAgent"
_PATH_XPC_SOURCES="../../../../../daemon/wifiNotifier/darwin/agent_xpc"

mkdir -p ${_OUT_FOLDER}
# ===================== COMPILING =======================
echo "[+] Compiling helper ..."
clang -Wall -O2 \
    -mmacosx-version-min=10.6 \
    -I${_PATH_XPC_SOURCES} \
		-framework Foundation -framework CoreLocation -framework CoreWLAN  -framework SystemConfiguration \
		-o ${_OUT_BINARY} main.m wifi.m ${_PATH_XPC_SOURCES}/xpc_client.m

if ! [ $? -eq 0 ]; then #check result of last command
  echo "FAILED"
  exit 1
fi

 echo "[ ] Done. Compiled binary: '${_BASE_DIR}/${_OUT_BINARY}'"

#daemon/wifiNotifier/darwin/agent_xpc
#ui/References/macOS/HelperProjects/launchAgent/