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

//go:build !fastping
// +build !fastping

package service

import (
	"fmt"
	"net"
	"sort"
	"time"

	"github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/helpers"
	"github.com/ivpn/desktop-app/daemon/ping"
	"github.com/ivpn/desktop-app/daemon/vpn"
)

// PingServers ping vpn servers.
//
// Pinging operation separated on few phases:
//  1. Fast ping: ping one host for each nearest location (locations only for specified VPN type when vpnTypePrioritized==true)
//     Operation ends after 'timeoutMs'. Then daemon sends response PingServersResp with all data were collected.
//  2. Full ping: Pinging all hosts for all locations. There is no time limit for this operation. It runs in background.
//     2.1) Ping all hosts only for specified VPN type (when vpnTypePrioritized==true) or for all VPN types (when vpnTypePrioritized==false)
//     2.1.1) Ping one host for all locations (for prioritized VPN protocol)
//     2.1.2) Ping the rest hosts for all locations (for prioritized VPN protocol)
//     2.2) (when vpnTypePrioritized==true) Ping all hosts for the rest protocols
//
// If pingAllHostsOnFirstPhase==true - daemon will ping all hosts for nearest locations on the phase (1)
// If skipSecondPhase==true - phase (2) will be skipped
//
// Additional info:
//
//	In some cases the multiple (and simultaneous pings) are leading to OS crash on macOS and Windows.
//	It happens when installed some third-party 'security' software.
//	Therefore, we using ping algorithm which avoids simultaneous pings and doing it one-by-one
func (s *Service) PingServers(timeoutMs int, vpnTypePrioritized vpn.Type, pingAllHostsOnFirstPhase bool, skipSecondPhase bool) (map[string]int, error) {

	if s._vpn != nil {
		return nil, fmt.Errorf("servers pinging skipped due to connected state")
	}
	if timeoutMs <= 0 {
		log.Debug("Servers pinging skipped: timeout argument value is 0")
		return nil, nil
	}

	// Block pinging when IVPNServersAccess==blocked
	if err := s.IsConnectivityBlocked(); err != nil {
		return nil, fmt.Errorf("servers pinging skipped: %v", err)
	}

	log.Info("Pinging servers...")
	timeoutTime := time.Now().Add(time.Millisecond * time.Duration(timeoutMs))

	var geoLocation *types.GeoLookupResponse = nil
	if timeoutMs >= 3000 {
		l, err := s._api.GeoLookup(1500)
		if err != nil {
			log.Warning("(pinging) unable to obtain geo-location (fastest server detection could be not accurate):", err)
		}
		geoLocation = l
	} else {
		log.Warning("(pinging) not enough time to check geo-location (fastest server detection could be not accurate)")
	}

	// do not allow multiple ping request simultaneously
	if !s._serversPingProgressSemaphore.TryAcquire(1) {
		return nil, fmt.Errorf("servers pinging skipped: ping already in progress, please try again after some delay")
	}
	defer func() {
		if skipSecondPhase {
			s._serversPingProgressSemaphore.Release(1)
		}
	}()

	// return value [host]latency
	result := make(map[string]int)

	allWgHosts, _ := s.getHostsToPing(geoLocation, false, vpn.WireGuard)
	allOvpnHosts, _ := s.getHostsToPing(geoLocation, false, vpn.OpenVPN)
	totalHosts := len(allWgHosts) + len(allOvpnHosts)

	startTime := time.Now()
	defer func(start time.Time) {
		log.Info(fmt.Sprintf("Fast ping finished in (%v): %d of %d pinged", time.Since(start), len(result), totalHosts))
	}(startTime)

	// 1) Fast ping: ping one host for each nearest location

	// get servers IP: IPs will be sorted by distance from current location (nearest - first)
	hosts, err := s.getHostsToPing(geoLocation, !pingAllHostsOnFirstPhase, vpnTypePrioritized)
	if err != nil {
		log.Info("Servers ping failed: " + err.Error())
		return nil, err
	}
	// First ping iteration. Doing it fast. 300ms max for each server
	s.pingIteration(hosts, result, 300, &timeoutTime)

	if !skipSecondPhase {
		// The first ping result already received.
		// So, now there is no rush to do second ping iteration. Doing it in background.
		//
		// 2) Full ping: Pinging all hosts for all locations. There is no time limit for this operation. It runs in background.
		go func() {
			defer func(start time.Time) {
				log.Info(fmt.Sprintf("Full ping finished in (%v): %d of %d pinged", time.Since(start), len(result), totalHosts))
			}(startTime)

			defer func() {
				if r := recover(); r != nil {
					log.Error("Panic in background ping: ", r)
					if err, ok := r.(error); ok {
						log.ErrorTrace(err)
					}
				}
				s._serversPingProgressSemaphore.Release(1)
			}()

			// FUNCTION to ping all hosts: the result is collected in 'result' map (defined earlier)
			doPingAllHosts := func(vpnType vpn.Type, onlyOneHostPerServer bool) {
				hosts, err = s.getHostsToPing(geoLocation, onlyOneHostPerServer, vpnType)
				if err != nil {
					log.Info("Servers ping failed: " + err.Error())
					return
				}

				s.pingIteration(hosts, result, 1000, nil)
				s._evtReceiver.OnPingStatus(result)
			}

			// All hosts (for prioritized VPN protocol when defined, or for all protocols (if vpnTypePrioritized not defined))

			// 2.1) Ping all hosts only for specified VPN type
			// 2.1.1) Ping one host for all locations (for prioritized VPN protocol)
			// 2.1.2) Ping the rest hosts for all locations (for prioritized VPN protocol)
			doPingAllHosts(vpnTypePrioritized, true)
			doPingAllHosts(vpnTypePrioritized, false)

			// 2.2) (when vpnTypePrioritized==true) Ping all hosts for the rest protocols
			if vpnTypePrioritized == vpn.OpenVPN {
				doPingAllHosts(vpn.WireGuard, false)
			} else if vpnTypePrioritized == vpn.WireGuard {
				doPingAllHosts(vpn.OpenVPN, false)
			}
		}()
	}

	// Return first ping result (fast ping: first stage)
	// This result may not contain results for all servers
	return result, nil
}

func (s *Service) pingIteration(hostsToPing []net.IP, pingedResult map[string]int, onePingTimeoutMs int, timeout *time.Time) /* map[string]int*/ {
	// OS-specific preparations (e.g. we need to add servers IPs to firewall exceptions list)
	if err := s.implPingServersStarting(hostsToPing); err != nil {
		log.Error("implPingServersStarting failed: " + err.Error())
	}

	defer func() {
		if err := s.implPingServersStopped(hostsToPing); err != nil {
			log.Error("implPingServersStopped failed: " + err.Error())
		}
	}()

	lastUpdateSentTime := time.Now()
	for _, h := range hostsToPing {
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

		// skip pinging twice same host
		if _, ok := pingedResult[ipStr]; ok {
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

		if stat.AvgRtt > 0 {
			ttl := int(stat.AvgRtt / time.Millisecond)
			pingedResult[ipStr] = ttl
		}

		if timeout == nil && time.Now().After(lastUpdateSentTime.Add(time.Second*2)) && len(pingedResult) > 0 {
			// periodically notify ping results when pinging in background
			s._evtReceiver.OnPingStatus(pingedResult)
			lastUpdateSentTime = time.Now()
		}
	}
}

// if 'currentLocation' defined - the output hosts list will be sorted by distance to current location
func (s *Service) getHostsToPing(currentLocation *types.GeoLookupResponse, onlyOneHostPerServer bool, vpnTypePrioritized vpn.Type) ([]net.IP, error) {
	// get servers info
	servers, err := s._serversUpdater.GetServers()
	if err != nil {
		return nil, fmt.Errorf("unable to get servers list: %w", err)
	}

	type hostInfo struct {
		Latitude  float32
		Longitude float32
		host      net.IP
	}

	hosts := make([]hostInfo, 0, uint16(len(servers.OpenvpnServers)+len(servers.WireguardServers)))

	// OpenVPN servers
	if vpnTypePrioritized != vpn.WireGuard {
		for _, s := range servers.OpenvpnServers {
			if len(s.Hosts) <= 0 {
				continue
			}

			for _, h := range s.Hosts {
				ip := net.ParseIP(h.Host)
				if ip != nil {
					hosts = append(hosts, hostInfo{Latitude: s.Latitude, Longitude: s.Longitude, host: ip})
				}
				if onlyOneHostPerServer {
					break
				}
			}
		}
	}

	// ping each WireGuard server
	if vpnTypePrioritized != vpn.OpenVPN {
		for _, s := range servers.WireguardServers {
			if len(s.Hosts) <= 0 {
				continue
			}

			for _, h := range s.Hosts {
				ip := net.ParseIP(h.Host)
				if ip != nil {
					hosts = append(hosts, hostInfo{Latitude: s.Latitude, Longitude: s.Longitude, host: ip})
				}
				if onlyOneHostPerServer {
					break
				}
			}
		}
	}

	if currentLocation != nil {
		cLat := float64(currentLocation.Latitude)
		cLot := float64(currentLocation.Longitude)
		sort.Slice(hosts, func(i, j int) bool {
			di := helpers.GetDistanceFromLatLonInKm(cLat, cLot, float64(hosts[i].Latitude), float64(hosts[i].Longitude))
			dj := helpers.GetDistanceFromLatLonInKm(cLat, cLot, float64(hosts[j].Latitude), float64(hosts[j].Longitude))
			return di < dj
		})
	}
	ret := make([]net.IP, 0, len(hosts))
	for _, h := range hosts {
		ret = append(ret, h.host)
	}
	return ret, nil
}
