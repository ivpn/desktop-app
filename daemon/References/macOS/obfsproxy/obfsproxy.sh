#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "${DIR}/obfsproxy"

exec arch -x86_64 /System/Library/Frameworks/Python.framework/Versions/2.6/bin/python ./obfsproxy.bin "$@"
