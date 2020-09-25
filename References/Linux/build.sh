#!/bin/bash

# Building electron app (result can be found here: dist_electron/ivpn-ui-XXX.AppImage)
# Configuring  app version in file 'package.json'

cd "$(dirname "$0")"
cd ../..
npm run electron:build