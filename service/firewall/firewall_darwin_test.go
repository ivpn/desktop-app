package firewall

import (
	"net"
	"testing"

	"github.com/ivpn/desktop-app-daemon/netinfo"
)

func TestGetNetworkInterfaces(t *testing.T) {
	inf, err := netinfo.InterfaceByIPAddr(net.ParseIP("127.0.0.1"))
	if err != nil {
		t.Error(err)
		return
	}

	if inf.Name != "lo0" {
		t.Error("Expected network interface: lo0")
	}
}
