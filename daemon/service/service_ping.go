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

// +build !fastping

package service

import (
	"fmt"
	"time"

	"github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/ping"
)

// PingServers ping vpn servers.
// In some cases the multiple (and simultaneous pings) are leading to OS crash on macOS and Windows.
// It happens when installed some third-party 'security' software.
// Therefore, we using ping algorithm which avoids simultaneous pings and doing it one-by-one
func (s *Service) PingServers(retryCount int, timeoutMs int) (map[string]int, error) {

	if s._vpn != nil {
		log.Info("Servers pinging skipped due to connected state")
		return nil, nil
	}

	if timeoutMs <= 0 {
		log.Debug("Servers pinging skipped: timeout argument value is 0")
		return nil, nil
	}

	timeoutTime := time.Now().Add(time.Millisecond * time.Duration(timeoutMs))

	var geoLocation *types.GeoLookupResponse = nil
	if timeoutMs >= 3000 {
		l, err := s._api.GeoLookup(1500)
		if err != nil {
			log.Warning("(pinging) unable to obtain geolocation (fastest server detection could be not accurate):", err)
		}
		geoLocation = l
	} else {
		log.Warning("(pinging) not enough time to check geolocation (fastest server detection could be not accurate)")
	}

	// get servers IP
	// IPs will be sorted by distance from current location (nearest - first)
	hosts, err := s.getHostsToPing(geoLocation)
	if err != nil {
		log.Info("Servers ping failed: " + err.Error())
		return nil, err
	}

	// OS-specific preparations (e.g. we need to add servers IPs to firewall exceptions list)
	if err := s.implIsGoingToPingServers(hosts); err != nil {
		log.Info("Servers ping failed : " + err.Error())
		return nil, err
	}

	result := make(map[string]int)

	funcPingIteration := func(onePingTimeoutMs int, timeout *time.Time) map[string]int {

		retMap := make(map[string]int)

		i := 0
		for _, h := range hosts {
			if s._vpn != nil {
				log.Info("Servers pinging stopped due to connected state")
				break
			}
			if timeout != nil && time.Now().Add(time.Millisecond*time.Duration(onePingTimeoutMs)).After(*timeout) {
				log.Info("Servers pinging stopped due max-timeout for this operation")
				break
			}

			if h == nil {
				continue
			}
			ipStr := h.String()
			if len(ipStr) <= 0 {
				continue
			}

			pinger, err := ping.NewPinger(ipStr)
			if err != nil {
				log.Error("Pinger creation error: " + err.Error())
				continue
			}

			pinger.SetPrivileged(true)
			pinger.Count = 1
			pinger.Timeout = time.Millisecond * time.Duration(onePingTimeoutMs)
			pinger.Run()
			stat := pinger.Statistics()
			i++

			if stat.AvgRtt > 0 {
				retMap[ipStr] = int(stat.AvgRtt / time.Millisecond)
			}

			if timeout == nil && len(retMap) > 0 && len(retMap)%10 == 0 {
				// periodically notify ping results when pinging in background
				s._evtReceiver.OnPingStatus(retMap)
			}
		}

		log.Info(fmt.Sprintf("Pinged %d of %d servers (%d successfully, timeout=%d)", i, len(hosts), len(retMap), onePingTimeoutMs))
		return retMap
	}

	// do not allow multiple ping request simultaneously
	if s._isServersPingInProgress {
		log.Info("Servers pinging skipped. Ping already in progress")
		return nil, nil
	}
	s._isServersPingInProgress = true

	// First ping iteration. Doing it fast. 300ms max for each server
	result = funcPingIteration(300, &timeoutTime)

	// The first ping result already received.
	// So, now there is no rush to do second ping iteration. Doing it in background.
	go func() {
		defer func() {
			s._isServersPingInProgress = false

			if r := recover(); r != nil {
				log.Error("Panic in background ping: ", r)
				if err, ok := r.(error); ok {
					log.ErrorTrace(err)
				}
			}
		}()

		ret := funcPingIteration(1000, nil)
		for k, v := range ret {
			if v <= 0 {
				continue
			}
			if oldVal, ok := result[k]; ok {
				if v < oldVal {
					result[k] = v
				}
			} else {
				result[k] = v
			}
		}
		s._evtReceiver.OnPingStatus(result)
	}()

	// Return first ping result
	// This result may not contain results for all servers
	return result, nil
}
