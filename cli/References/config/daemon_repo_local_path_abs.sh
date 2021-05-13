#!/bin/bash

# Script prints absolute path to local  repository of IVPN Daemon sources
# It reads relative path info from 'config/daemon_repo_local_path.txt'
# How to use in subscripts:
#   DAEMON_REPO_ABS_PATH=$("./daemon_repo_local_path_abs.sh")

# Exit immediately if a command exits with a non-zero status.
set -e

cd "$(dirname "$0")"
RELATIVE_PATH=$(<'daemon_repo_local_path.txt')
cd $RELATIVE_PATH

pwd
