//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app-daemon
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
//
//  This file is part of the Daemon for IVPN Client Desktop.
//
//  The Daemon for IVPN Client Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The Daemon for IVPN Client Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the Daemon for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
//

package firewall_test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/ivpn/desktop-app-daemon/ping"
	"github.com/ivpn/desktop-app-daemon/service/firewall"
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
