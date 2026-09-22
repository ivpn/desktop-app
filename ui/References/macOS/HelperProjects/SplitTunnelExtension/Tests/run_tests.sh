#!/bin/bash
# Builds and runs the standalone tests for the Split Tunnel extension's
# decision logic. Needs only the command line tools; run from anywhere.
set -euo pipefail
cd "$(dirname "$0")"
OUT="$(mktemp -d)"
trap 'rm -rf "$OUT"' EXIT
clang -fobjc-arc -Wall -Wextra -Werror -mmacosx-version-min=12.0 \
      -framework Foundation -framework NetworkExtension -framework Network -lbsm \
      ../STPathMatching.m ../STPhysicalInterfaceSelector.m ../STLog.m \
      STPathMatchingTests.m STDefaultRouteTests.m -o "$OUT/tests"
"$OUT/tests"
