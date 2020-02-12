package firewall_test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/ivpn/desktop-app-daemon/service/firewall"

	//go get github.com/sparrc/go-ping
	"github.com/sparrc/go-ping"
)

func CheckConnectivity() bool {
	pinger, err := ping.NewPinger("8.8.8.8")
	// Windows Support: You must use pinger.SetPrivileged(true), otherwise you will receive an error
	pinger.SetPrivileged(true)

	if err != nil {
		panic(err)
	}

	pinger.Count = 1
	pinger.Timeout = time.Second
	pinger.Run()                 // blocks until finished
	stats := pinger.Statistics() // get send/receive/rtt stats
	return stats.PacketsRecv > 0
}

func TestEnableDisable(t *testing.T) {

	// Disable (after test finished)
	defer func() {
		firewall.SetEnabled(false)
	}()

	// Check FW initial  state
	isEnabled, err := firewall.GetEnabled()
	if err != nil {
		t.Error(err)
		return
	}
	if isEnabled != false {
		t.Error("Test failed: initial firewall state should be disabled")
		return
	}
	if CheckConnectivity() == false {
		t.Error("Test failed: no connectivity")
		return
	}

	// Disable
	if err = firewall.SetEnabled(false); err != nil {
		t.Error(err)
		return
	}
	//Check
	isEnabled, err = firewall.GetEnabled()
	if err != nil {
		t.Error(err)
		return
	}
	if isEnabled != false {
		t.Error("Test failed: expected FW state: false")
		return
	}
	if CheckConnectivity() == false {
		t.Error("Test failed: no connectivity")
		return
	}

	// Enable
	if err = firewall.SetEnabled(true); err != nil {
		t.Error(err)
		return
	}
	//Check
	isEnabled, err = firewall.GetEnabled()
	if err != nil {
		t.Error(err)
		return
	}
	if isEnabled != true {
		t.Error("Test failed: expected FW state: true")
		return
	}
	if CheckConnectivity() == true {
		t.Error("Test failed: available connectivity but expected to be blocked")
		return
	}

	// Disable
	if err = firewall.SetEnabled(false); err != nil {
		t.Error(err)
		return
	}
	//Check
	isEnabled, err = firewall.GetEnabled()
	if err != nil {
		t.Error(err)
		return
	}
	if isEnabled != false {
		t.Error("Test failed: expected FW state: false")
		return
	}
	if CheckConnectivity() == false {
		t.Error("Test failed: no connectivity")
		return
	}
}

func TestClientConnected(t *testing.T) {

	// Disable (after test finished)
	defer func() {
		firewall.SetEnabled(false)
	}()

	// Check FW initial  state
	isEnabled, err := firewall.GetEnabled()
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println(isEnabled)

	err = firewall.ClientConnected(net.IPv4(10, 59, 44, 2))
	if err != nil {
		t.Error(err)
		return
	}

	err = firewall.ClientDisconnected()
	if err != nil {
		t.Error(err)
		return
	}
	err = firewall.ClientConnected(net.IPv4(10, 59, 44, 2))
	if err != nil {
		t.Error(err)
		return
	}

}
