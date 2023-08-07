#!/bin/sh

# Exit immediately if a command exits with a non-zero status.
set -e

cd "$(dirname "$0")"
BASE_DIR="$(pwd)" #set base folder of script location


BUILD_DIR=${BASE_DIR}/../_deps/kem-helper # work directory

echo "******** Creating work-folder (${BUILD_DIR})..."
rm -rf ${BUILD_DIR}
mkdir -pv ${BUILD_DIR}

echo "******** Compiling (kem-helper)..."
./../../common/kem-helper/build.sh -d $BUILD_DIR

echo "********************************"
echo "******** BUILD COMPLETE ********"
echo "********************************"