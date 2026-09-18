//go:build darwin
// +build darwin

package service

import "testing"

func TestMacAppBundlePath(t *testing.T) {
	tests := []struct {
		in       string
		expected string
	}{
		{"/Applications/Firefox.app", "/Applications/Firefox.app"},
		{"/Applications/Firefox.app/", "/Applications/Firefox.app"},
		{"/Applications/Firefox.app/Contents/MacOS/firefox", "/Applications/Firefox.app"},
		// Nested bundles resolve to the innermost one (that is the identity the user picked).
		{"/Applications/Foo.app/Contents/Helpers/Bar.app/Contents/MacOS/bar", "/Applications/Foo.app/Contents/Helpers/Bar.app"},
		// No '.app' ancestor: returned unchanged, matching the extension's exact-path fallback.
		{"/usr/local/bin/mytool", "/usr/local/bin/mytool"},
		{"/", "/"},
		{"", ""},
	}

	for _, test := range tests {
		if got := macAppBundlePath(test.in); got != test.expected {
			t.Errorf("macAppBundlePath(%q) = %q, expected %q", test.in, got, test.expected)
		}
	}
}

func TestIsMacOwnAppBundlePath(t *testing.T) {
	tests := []struct {
		in       string
		expected bool
	}{
		{"/Applications/IVPN.app", true},
		{"/Applications/IVPN.app/Contents/MacOS/IVPN Agent", true},
		{"/Applications/IVPN.app.bak", false},
		{"/Applications/IVPN.appendix.app", false},
		{"/Applications/Firefox.app", false},
		{"", false},
	}

	for _, test := range tests {
		if got := isMacOwnAppBundlePath(test.in); got != test.expected {
			t.Errorf("isMacOwnAppBundlePath(%q) = %v, expected %v", test.in, got, test.expected)
		}
	}
}
