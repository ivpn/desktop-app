//go:build darwin
// +build darwin

package oshelpers

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework Foundation

#include <stdlib.h>

int app_bundle_info(const char *bundlePath, char **outDisplayName, char **outExecutableName);
int app_icon_png(const char *bundlePath, int maxSizePx, unsigned char **outData, long *outLen);
void app_free(void *ptr);
*/
import "C"

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"
)

// macOS folders commonly containing application bundles. "~/Applications" is
// appended below once the logged-in user's HOME is known (the daemon runs as
// root, so it can't rely on its own HOME).
var appSearchDirs = []string{
	"/Applications",
	"/Applications/Utilities",
	"/System/Applications",
	"/System/Applications/Utilities",
}

// Downscaled to keep the protocol payload small - matches the size the
// Settings UI actually renders app icons at.
const appIconMaxSizePx = 64

type extraArgsGetInstalledApps struct {
	EnvVar_HOME string
}

func implGetInstalledApps(extraArgsJSON string) ([]AppInfo, error) {
	home := ""
	if len(extraArgsJSON) > 0 {
		var extraArgs extraArgsGetInstalledApps
		if err := json.Unmarshal([]byte(extraArgsJSON), &extraArgs); err == nil {
			home = extraArgs.EnvVar_HOME
		}
	}

	dirs := append([]string(nil), appSearchDirs...)
	if len(home) > 0 {
		dirs = append(dirs, filepath.Join(home, "Applications"))
	}

	selfBundlePath := ownAppBundlePath()

	retValues := make([]AppInfo, 0, 64)
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue // folder may legitimately not exist (e.g. no ~/Applications)
		}
		for _, e := range entries {
			if !e.IsDir() || !strings.HasSuffix(e.Name(), ".app") {
				continue
			}
			bundlePath := filepath.Join(dir, e.Name())
			if len(selfBundlePath) > 0 && bundlePath == selfBundlePath {
				continue // never offer IVPN's own app as a Split-Tunnel candidate
			}

			displayName, _ := appBundleInfo(bundlePath)
			if len(displayName) == 0 {
				displayName = strings.TrimSuffix(e.Name(), ".app")
			}
			// AppBinaryPath is the '.app' bundle path, not an inner executable:
			// the extension matches by bundle-path prefix, so this is what
			// Preferences.SplitTunnelApps must store.
			retValues = append(retValues, AppInfo{AppName: displayName, AppBinaryPath: bundlePath})
		}
	}

	return retValues, nil
}

func implGetFunc_BinaryIconBase64() func(binaryPath string) (icon string, err error) {
	return getBinaryIconBase64
}

func getBinaryIconBase64(binaryPath string) (string, error) {
	pngData, err := appIconPNG(binaryPath, appIconMaxSizePx)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(pngData), nil
}

// appBundleInfo reads CFBundleDisplayName/CFBundleName (falling back to the
// executable name) via NSBundle, which transparently handles both XML and
// binary plist formats - no plist parser of our own needed.
func appBundleInfo(bundlePath string) (displayName string, executableName string) {
	cBundlePath := C.CString(bundlePath)
	defer C.free(unsafe.Pointer(cBundlePath))

	var cDisplayName, cExecutableName *C.char
	if C.app_bundle_info(cBundlePath, &cDisplayName, &cExecutableName) != 0 {
		return "", ""
	}
	if cDisplayName != nil {
		displayName = C.GoString(cDisplayName)
		C.app_free(unsafe.Pointer(cDisplayName))
	}
	if cExecutableName != nil {
		executableName = C.GoString(cExecutableName)
		C.app_free(unsafe.Pointer(cExecutableName))
	}
	return displayName, executableName
}

func appIconPNG(bundlePath string, maxSizePx int) ([]byte, error) {
	cBundlePath := C.CString(bundlePath)
	defer C.free(unsafe.Pointer(cBundlePath))

	var cData *C.uchar
	var cLen C.long
	if C.app_icon_png(cBundlePath, C.int(maxSizePx), &cData, &cLen) != 0 || cData == nil {
		return nil, fmt.Errorf("unable to extract icon for '%s'", bundlePath)
	}
	defer C.app_free(unsafe.Pointer(cData))

	return C.GoBytes(unsafe.Pointer(cData), C.int(cLen)), nil
}

// ownAppBundlePath returns the ".app" bundle containing the currently running
// daemon binary (e.g. "/Applications/IVPN.app"), or "" if the daemon isn't
// running from inside one (e.g. a dev build) - used to make sure IVPN's own
// app is never offered as a Split-Tunnel candidate.
func ownAppBundlePath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	for dir := filepath.Dir(exe); dir != "/" && dir != "."; {
		if strings.HasSuffix(dir, ".app") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
