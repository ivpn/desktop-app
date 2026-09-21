#!/bin/bash
# Builds and runs the standalone tests for the Split Tunnel extension's
# decision logic. Needs only the command line tools; run from anywhere.
set -euo pipefail
cd "$(dirname "$0")"
OUT="$(mktemp -d)"
trap 'rm -rf "$OUT"' EXIT
clang -fobjc-arc -Wall -Wextra -Werror \
      -framework Foundation -framework NetworkExtension -lbsm \
      ../STPathMatching.m STPathMatchingTests.m -o "$OUT/tests"
"$OUT/tests"
