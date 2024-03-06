#!/bin/bash

error() {
  echo "Error: $1"
  exit 1
}

clean() {
  echo "Cleaning up..."
  sudo launchctl unload -w /Library/LaunchDaemons/ivpn.xpc.test.service.plist || echo "No service to unload"
  sudo rm /Library/LaunchDaemons/ivpn.xpc.test.service.plist /tmp/xpc_server || echo "No files to remove"
  rm -rf _out || echo "No folder to remove"
}

clean
# if "-clean" is passed as argument, just clean the environment and exit
if [ "$1" == "-clean" ]; then
  exit 0
fi

# Create output folder
mkdir -p _out || error 'Output folder creation failed'

# Compile the server & client
gcc  -x objective-c -framework Foundation xpc_server_main.c ../xpc_server.m -o _out/xpc_server || error 'Server compilation failed'
gcc  -x objective-c -framework Foundation xpc_client_main.c ../xpc_client.m -o _out/xpc_client || error 'Client compilation failed'

# Save server on it's location
cp _out/xpc_server /tmp

# Load daemon
sudo cp ivpn.xpc.test.service.plist /Library/LaunchDaemons
sudo launchctl load -w /Library/LaunchDaemons/ivpn.xpc.test.service.plist || error '"launchctl load" failed'

# Call client
./_out/xpc_client || error 'Client start failed'

clean