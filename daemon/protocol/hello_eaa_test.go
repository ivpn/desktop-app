package protocol

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ivpn/desktop-app/daemon/protocol/eaa"
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

func TestNotifyClientsRedactsUnauthenticatedHello(t *testing.T) {
	secretFile := filepath.Join(t.TempDir(), "eaa")
	if err := os.WriteFile(secretFile, []byte("hash"), 0600); err != nil {
		t.Fatal(err)
	}

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	p := &Protocol{
		_connections: map[net.Conn]*connectionInfo{
			server: {IsAuthenticated: false},
		},
		_eaa: eaa.Init(secretFile),
	}

	hello := &types.HelloResp{}
	hello.Session.AccountID = "acct"
	hello.Session.Session = "session-token"

	lineCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		line, err := bufio.NewReader(client).ReadString('\n')
		if err != nil {
			errCh <- err
			return
		}
		lineCh <- line
	}()

	p.notifyClients(hello)

	var line string
	select {
	case line = <-lineCh:
	case err := <-errCh:
		t.Fatal(err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Hello")
	}

	var msg struct {
		Session struct {
			AccountID string
			Session   string
		}
	}
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Session.AccountID != "" || msg.Session.Session != "" {
		t.Fatalf("wire session AccountID=%q Session=%q", msg.Session.AccountID, msg.Session.Session)
	}
	if hello.Session.Session != "session-token" {
		t.Fatal("notify mutated the original Hello")
	}
}
