package winlib_test

import (
	"ivpn/daemon/service/firewall/winlib"
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/sparrc/go-ping"
)

func TestBlockAll(t *testing.T) {

	var mgr winlib.Manager

	var providerKey = syscall.GUID{Data1: 0xfed0afd4, Data2: 0x98d4, Data3: 0x4233, Data4: [8]byte{0xa4, 0xf3, 0x8b, 0x7c, 0x02, 0x44, 0x50, 0x01}}
	var sublayerKey = syscall.GUID{Data1: 0xfed0afd4, Data2: 0x98d4, Data3: 0x4233, Data4: [8]byte{0xa4, 0xf3, 0x8b, 0x7c, 0x02, 0x44, 0x50, 0x02}}

	const filterDName = "IVPN Test"
	const filterDDesc = "IVPN Test filter"

	v4Layers := []syscall.GUID{winlib.FwpmLayerAleAuthConnectV4, winlib.FwpmLayerAleAuthRecvAcceptV4}
	v6Layers := []syscall.GUID{winlib.FwpmLayerAleAuthConnectV6, winlib.FwpmLayerAleAuthRecvAcceptV6}

	isPingableFunc := func(ip net.IP) bool {
		pinger, err := ping.NewPinger(ip.String())
		if err != nil {
			return false
		}

		pinger.SetPrivileged(true)
		pinger.Count = 3
		pinger.Interval = time.Microsecond
		pinger.Timeout = time.Second

		pinger.Run()

		stat := pinger.Statistics()
		if stat.PacketsRecv > 0 {
			return true
		}
		return false
	}

	enableFunc := func() {
		mgr.TransactionStart()
		defer mgr.TransactionCommit()

		provider := winlib.CreateProvider(providerKey, "IVPN Test", "IVPN Test WFP Provider", false)
		sublayer := winlib.CreateSubLayer(sublayerKey, providerKey, "IVPN Test", "IVPN Test WFP Sublayer", 2300, false)

		pinfo, err := mgr.GetProviderInfo(providerKey)
		if err != nil {
			t.Error(err)
		}

		if !pinfo.IsInstalled {
			if err = mgr.AddProvider(provider); err != nil {
				t.Error(err)
			}
		}

		installed, err := mgr.IsSubLayerInstalled(sublayerKey)
		if err != nil {
			t.Error(err)
		}

		if !installed {
			if err = mgr.AddSubLayer(sublayer); err != nil {
				t.Error(err)
			}
		}

		for _, l := range v6Layers {
			_, err := mgr.AddFilter(winlib.NewFilterBlockAll(providerKey, l, sublayerKey, filterDName, filterDDesc, true, false))
			if err != nil {
				t.Error(err)
			}
		}

		for _, l := range v4Layers {
			_, err := mgr.AddFilter(winlib.NewFilterBlockAll(providerKey, l, sublayerKey, filterDName, filterDDesc, false, false))
			if err != nil {
				t.Error(err)
			}
		}

	}

	disableAllFunc := func() {
		if err := mgr.TransactionAbort(); err != nil {
			t.Error(err)
		}

		if err := mgr.TransactionStart(); err != nil {
			t.Error(err)
		}
		defer mgr.TransactionCommit()

		for _, l := range v6Layers {
			if err := mgr.DeleteFilterByProviderKey(providerKey, l); err != nil {
				t.Error(err)
			}
		}

		for _, l := range v4Layers {
			if err := mgr.DeleteFilterByProviderKey(providerKey, l); err != nil {
				t.Error(err)
			}
		}

		installed, err := mgr.IsSubLayerInstalled(sublayerKey)
		if err != nil {
			t.Error(err)
		}
		if installed {
			if err := mgr.DeleteSubLayer(sublayerKey); err != nil {
				t.Error(err)
			}
		}

		pinfo, err := mgr.GetProviderInfo(providerKey)
		if err != nil {
			t.Error(err)
		}
		if pinfo.IsInstalled {
			if err := mgr.DeleteProvider(providerKey); err != nil {
				t.Error(err)
			}
		}
	}

	defer func() {
		disableAllFunc()
	}()

	disableAllFunc()
	// PING
	if isPingableFunc(net.IPv4(1, 1, 1, 1)) == false {
		t.Error("Ping NOT received (2)")
	}

	for i := 0; i < 3; i++ {
		// PING
		if isPingableFunc(net.IPv4(1, 1, 1, 1)) == false {
			t.Error("Ping NOT received (3)")
		}

		enableFunc()

		// PING
		if isPingableFunc(net.IPv4(1, 1, 1, 1)) == true {
			t.Error("Ping received (4)")
		}

		disableAllFunc()

		// PING
		if isPingableFunc(net.IPv4(1, 1, 1, 1)) == false {
			t.Error("Ping NOT received (5)")
		}
	}

}
