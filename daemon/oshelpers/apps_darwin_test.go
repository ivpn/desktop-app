//go:build darwin
// +build darwin

package oshelpers

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

// Minimal smoke test for the cgo bridge to NSBundle/NSWorkspace: confirms the
// daemon can actually enumerate installed apps and extract an icon at
// runtime, not just compile. Does not assert on which apps are installed
// (that varies by machine), only that the mechanism works end to end.
func TestGetInstalledApps_Smoke(t *testing.T) {
	apps, err := GetInstalledApps("")
	if err != nil {
		t.Fatalf("GetInstalledApps failed: %v", err)
	}
	if len(apps) == 0 {
		t.Fatal("GetInstalledApps returned no apps - expected at least the system-bundled ones")
	}
	for _, a := range apps {
		if len(a.AppName) == 0 {
			t.Errorf("app with empty AppName: %+v", a)
		}
		if len(a.AppBinaryPath) == 0 {
			t.Errorf("app with empty AppBinaryPath: %+v", a)
		}
	}

	if !IsCanGetAppIconForBinary() {
		t.Fatal("IsCanGetAppIconForBinary() returned false - expected true on macOS")
	}
	icon, err := GetBinaryIconBase64(apps[0].AppBinaryPath)
	if err != nil {
		t.Fatalf("GetBinaryIconBase64(%q) failed: %v", apps[0].AppBinaryPath, err)
	}
	// The UI binds this straight into an <img src>, so the data-URL prefix is
	// part of the contract, not decoration.
	const pngDataURLPrefix = "data:image/png;base64,"
	payload, ok := strings.CutPrefix(icon, pngDataURLPrefix)
	if !ok {
		t.Fatalf("GetBinaryIconBase64(%q) is not prefixed with %q", apps[0].AppBinaryPath, pngDataURLPrefix)
	}
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("GetBinaryIconBase64(%q) payload is not valid base64: %v", apps[0].AppBinaryPath, err)
	}
	if !bytes.HasPrefix(raw, []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatalf("GetBinaryIconBase64(%q) payload is not a PNG", apps[0].AppBinaryPath)
	}
}
