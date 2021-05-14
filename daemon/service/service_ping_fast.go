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

// +build fastping

package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/ivpn/desktop-app/daemon/ping"
)

// PingServers ping vpn servers
func (s *Service) PingServers(retryCount int, timeoutMs int) (map[string]int, error) {

	// do not allow multiple ping request simultaneously
	if s._isServersPingInProgress {
		log.Info("Servers pinging skipped. Ping already in progress")
		return nil, nil
	}
	defer func() { s._isServersPingInProgress = false }()
	s._isServersPingInProgress = true

	vpn := s._vpn
	if vpn != nil {
		log.Info("Servers pinging skipped due to connected state")
		return nil, nil
	}

	if retryCount <= 0 || timeoutMs <= 0 {
		log.Debug("Servers pinging skipped: arguments value is 0")
		return nil, nil
	}

	// get servers IP
	hosts, err := s.getHostsToPing(nil)
	if err != nil {
		log.Info("Servers ping failed: " + err.Error())
		return nil, err
	}

	// OS-specific preparations (e.g. we need to add servers IPs to firewall exceptions list)
	if err := s.implIsGoingToPingServers(hosts); err != nil {
		log.Info("Servers ping failed : " + err.Error())
		return nil, err
	}

	// initialize waiter (will wait to finish all go-routines)
	var waiter sync.WaitGroup

	type pair struct {
		host string
		ping int
	}

	resultChan := make(chan pair, 1)
	// define generic ping function
	pingFunc := func(ip string) {
		// notify waiter: goroutine is finished
		defer waiter.Done()

		pinger, err := ping.NewPinger(ip)
		if err != nil {
			log.Error("Pinger creation error: " + err.Error())
			return
		}

		pinger.SetPrivileged(true)
		pinger.Count = retryCount
		pinger.Interval = time.Millisecond * 1000 // do not use small interval (<350ms). Possible unexpected behavior: pings never return sometimes
		pinger.Timeout = time.Millisecond * time.Duration(timeoutMs)

		pinger.Run()

		stat := pinger.Statistics()

		// Pings filtering ...
		// there is a chance that one ping responce is much higher than the rest received responses
		// This, for example, observed on some virtual machines. The first ping result is catastrophically higher than the rest
		// Hera we are ignoring such situations (ignoring highest pings when necessary)
		var avgPing time.Duration = 0
		maxAllowedTTL := float32(stat.AvgRtt) * 1.3
		if stat.PacketLoss < 0 || float32(stat.MaxRtt) < maxAllowedTTL {
			avgPing = stat.AvgRtt
			//log.Debug(int(stat.AvgRtt/time.Millisecond), " == ", int(avgPing/time.Millisecond), "\t", stat)
		} else {
			cntResults := 0
			for _, p := range stat.Rtts {
				if float32(p) >= maxAllowedTTL {
					continue
				}
				avgPing += p
				cntResults++
			}
			if cntResults > 0 {
				avgPing = avgPing / time.Duration(cntResults)
			} else {
				avgPing = stat.AvgRtt
			}
			//log.Debug(int(stat.AvgRtt/time.Millisecond), " -> ", int(avgPing/time.Millisecond), "\t", stat)
		}

		resultChan <- pair{host: ip, ping: int(avgPing / time.Millisecond)}

		// ... pings filtering

		// Original pings data:
		//resultChan <- pair{host: ip, ping: int(stat.AvgRtt / time.Millisecond)}
	}

	log.Info("Pinging servers...")
	for _, s := range hosts {
		if s == nil {
			continue
		}
		ipStr := s.String()
		if len(ipStr) <= 0 {
			continue
		}
		waiter.Add(1) // +1 goroutine to wait
		go pingFunc(ipStr)
	}

	successfullyPinged := 0
	retMap := make(map[string]int)
	done := make(chan bool)
	go func() {
		for {
			select {
			case r := <-resultChan:
				retMap[r.host] = r.ping
				if r.ping > 0 {
					successfullyPinged = successfullyPinged + 1
				}
			case <-done:
				return
			}
		}
	}()

	waiter.Wait()
	done <- true

	log.Info(fmt.Sprintf("Pinged %d of %d servers (%d successfully)", len(retMap), len(hosts), successfullyPinged))

	return retMap, nil
}
