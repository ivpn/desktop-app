package firewall

import (
	"fmt"
	"ivpn/daemon/service/netinfo"
	"net"
	"testing"
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

func TestGetLocalIPs(t *testing.T) {
	fmt.Println(implGetLocalIPs())
}
