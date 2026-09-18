//go:build darwin
// +build darwin

package splittun

import "testing"

// The extension is driven entirely by the Electron main process off the
// existing SplitTunnelStatus fields, so implApplyConfig has nothing left to
// resolve/store - this only guards against a regression reintroducing a
// fatal path here.
func TestApplyConfig_NoOp(t *testing.T) {
	if err := implApplyConfig(false, false, false, false, ConfigAddresses{}, nil); err != nil {
		t.Fatalf("implApplyConfig failed: %v", err)
	}
	if err := implApplyConfig(true, true, true, true, ConfigAddresses{}, []string{"/Applications/Firefox.app"}); err != nil {
		t.Fatalf("implApplyConfig failed: %v", err)
	}
}
