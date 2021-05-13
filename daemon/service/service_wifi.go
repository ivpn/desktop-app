//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
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

package service

import (
	"time"

	"github.com/ivpn/desktop-app-daemon/wifiNotifier"
)

type wifiInfo struct {
	ssid       string
	isInsecure bool
}

var lastWiFiInfo *wifiInfo
var timerDelayedNotify *time.Timer

const delayBeforeWiFiChangeNotify = time.Second * 1

func (s *Service) initWiFiFunctionality() (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("initWiFiFunctionality PANIC (recovered): ", r)
		}
	}()

	return wifiNotifier.SetWifiNotifier(s.onWiFiChanged)
}

func (s *Service) onWiFiChanged(ssid string) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("onWiFiChanged PANIC (recovered): ", r)
		}
	}()

	isInsecure := wifiNotifier.GetCurrentNetworkIsInsecure()

	lastWiFiInfo = &wifiInfo{
		ssid,
		isInsecure}

	// do delay before processing wifi change
	// (same wifi change event can occur several times in short period of time)
	if timerDelayedNotify != nil {
		timerDelayedNotify.Stop()
		timerDelayedNotify = nil
	}
	timerDelayedNotify = time.AfterFunc(delayBeforeWiFiChangeNotify, func() {
		if lastWiFiInfo == nil || lastWiFiInfo.ssid != ssid || lastWiFiInfo.isInsecure != isInsecure {
			return // do nothing (new wifi info available)
		}

		// notify clients about WiFi change
		s._evtReceiver.OnWiFiChanged(ssid, isInsecure)
	})
}

// GetWiFiCurrentState returns info about currently connected wifi
func (s *Service) GetWiFiCurrentState() (ssid string, isInsecureNetwork bool) {
	return wifiNotifier.GetCurrentSSID(), wifiNotifier.GetCurrentNetworkIsInsecure()
}

// GetWiFiAvailableNetworks returns list of available WIFI networks
func (s *Service) GetWiFiAvailableNetworks() []string {
	return wifiNotifier.GetAvailableSSIDs()
}
