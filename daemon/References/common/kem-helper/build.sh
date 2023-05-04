#!/bin/bash

# #############################################################
# Dependencies: such packages required to be installed: 
# sudo apt install -y astyle cmake gcc ninja-build libssl-dev python3-pytest python3-pytest-xdist unzip xsltproc doxygen graphviz python3-yaml valgrind
# #############################################################

_SCRIPT_DIR="$( cd "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
_WORK_FOLDER=$_SCRIPT_DIR/_out_linux

set -e

# Error handling function
handle_error() {
    echo "[!] An ERROR occurred in the script!"
    echo "    Please, note that script has dependencies (sudo apt install -y astyle git cmake gcc ninja-build libssl-dev python3-pytest python3-pytest-xdist unzip xsltproc doxygen graphviz python3-yaml valgrind)"
    echo "    Exiting."
    exit 1
}
# Set the trap to catch errors
trap 'handle_error' ERR

# reading destination folder from arguments
while getopts ":d:" opt; do
  case $opt in
    d)  _WORK_FOLDER="$OPTARG"
        if [ ! -d "$_WORK_FOLDER" ]; then 
            echo "[!] ERROR: '$_WORK_FOLDER' does not exists!"
            exit 1
        fi
    ;;    
  esac
done
_WORK_FOLDER=$(realpath "${_WORK_FOLDER}")

echo "[i] Using work folder: $_WORK_FOLDER"
_OUT_FOLDER=$_WORK_FOLDER/kem-helper-bin
_LIBOQS_FOLDER=$_WORK_FOLDER/liboqs
_LIBOQS_SOURCES_FOLDER=$_LIBOQS_FOLDER/liboqs
_LIBOQS_INSTALL_FOLDER=$_LIBOQS_FOLDER/INSTALL

if [ ! -d $_LIBOQS_FOLDER ]; then 
    echo "[*] Creating '$_LIBOQS_FOLDER' ..."
    mkdir -p $_LIBOQS_FOLDER
else
    echo "[*] Erasing '$_LIBOQS_FOLDER' ..."
    rm -fr $_LIBOQS_FOLDER/*
fi 
cd $_LIBOQS_FOLDER

echo "[*] Gettings liboqs sources ..."

git clone --depth 1 -b main https://github.com/open-quantum-safe/liboqs.git
cd liboqs

echo "[*] Configuring and compiling liboqs ..."
mkdir build && cd build

cmake -GNinja .. \
        -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_INSTALL_PREFIX=$_LIBOQS_INSTALL_FOLDER \
        -DOQS_BUILD_ONLY_LIB=ON \
        -DBUILD_SHARED_LIBS=OFF \
        -DOQS_USE_OPENSSL=OFF \
        -DOQS_DIST_BUILD=ON 
ninja
ninja install 

echo "[*] Compiling kem-helper ..."

if [ ! -d $_OUT_FOLDER ]; then 
    echo "[*] Creating '$_OUT_FOLDER' ..."
    mkdir -p $_OUT_FOLDER
else
    echo "[*] Erasing '$_OUT_FOLDER' ..."
    rm -fr $_OUT_FOLDER/*
fi 
echo "Sources '$_SCRIPT_DIR'" > $_OUT_FOLDER/readme.md

# Change the current working directory to the location of the source files
cd $_SCRIPT_DIR

gcc main.c base64.c -o $_OUT_FOLDER/kem-helper -Wall -O2 -I$_LIBOQS_INSTALL_FOLDER/include -L$_LIBOQS_INSTALL_FOLDER/lib -loqs -Wl,-z,stack-size=5242880

echo "[ ] SUCCESS"
echo "    kem-helper binary: '$_OUT_FOLDER/kem-helper'"