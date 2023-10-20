//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 IVPN Limited.
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

package service

import (
	"time"

	"github.com/ivpn/desktop-app/daemon/wifiNotifier"
)

var timerDelayedWifiNotify *time.Timer

const delayBeforeWiFiChangeNotify = time.Second * 1

func (s *Service) initWiFiFunctionality() (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("initWiFiFunctionality PANIC (recovered): ", r)
		}
	}()

	return wifiNotifier.SetWifiNotifier(s.onWiFiChanged)
}

func (s *Service) onWiFiChanged() {
	defer func() {
		if r := recover(); r != nil {
			log.Error("onWiFiChanged PANIC (recovered): ", r)
		}
	}()

	// Stop old postponed notifier call
	oldTimerId := timerDelayedWifiNotify
	timerDelayedWifiNotify = nil
	if oldTimerId != nil {
		oldTimerId.Stop()
	}

	// Delay before processing wifi change
	// (same wifi change event can occur several times in short period of time)
	timerDelayedWifiNotify = time.AfterFunc(delayBeforeWiFiChangeNotify, func() {

		info, err := wifiNotifier.GetCurrentWifiInfo()
		if err != nil {
			log.Error("Can not obtain current WiFi info: ", err)
			return
		}

		// notify clients about WiFi change
		s._evtReceiver.OnWiFiChanged(info)

		// 'trusted-wifi' functionality: auto-connect if necessary
		s.autoConnectIfRequired(OnWifiChanged, &info)
	})
}

// GetWiFiCurrentState returns info about currently connected wifi
func (s *Service) GetWiFiCurrentState() wifiNotifier.WifiInfo {
	info, _ := wifiNotifier.GetCurrentWifiInfo()
	return info
}

// GetWiFiAvailableNetworks returns list of available WIFI networks
func (s *Service) GetWiFiAvailableNetworks() []string {
	return wifiNotifier.GetAvailableSSIDs()
}
