package protocol

import (
	"testing"

	"github.com/ivpn/desktop-app/daemon/protocol/types"
)

func TestHelloForClientOmitsSessionUntilEAA(t *testing.T) {
	full := &types.HelloResp{}
	full.Session.AccountID = "acct"
	full.Session.Session = "session-token"
	full.ParanoidMode.IsEnabled = true

	got := helloForClient(full, false, true).(*types.HelloResp)
	if got.Session.Session != "" || got.Session.AccountID != "" {
		t.Fatalf("unauthenticated Hello kept session AccountID=%q Session=%q", got.Session.AccountID, got.Session.Session)
	}
	if full.Session.Session != "session-token" {
		t.Fatal("redaction mutated the original Hello")
	}

	authed := helloForClient(full, true, true).(*types.HelloResp)
	if authed.Session.Session != "session-token" {
		t.Fatalf("authenticated Hello Session = %q", authed.Session.Session)
	}

	disabled := helloForClient(full, false, false).(*types.HelloResp)
	if disabled.Session.Session != "session-token" {
		t.Fatalf("EAA-disabled Hello Session = %q", disabled.Session.Session)
	}
}
