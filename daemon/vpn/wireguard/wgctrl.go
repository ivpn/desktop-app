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

package wireguard

import (
	"fmt"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl"
)

// WaitForFirstHanshake waits for a handshake during 'timeout' time.
// if isStopArray is defined and at lease one of it's elements == true: function stops and channel closes
func WaitForWireguardFirstHanshakeChan(tunnelName string, isStopArray []*bool, logFunc func(string)) <-chan error {
	retChan := make(chan error, 1)

	go func() (retError error) {
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(error); ok {
					retChan <- fmt.Errorf("crash (recovered): %w", err)
				}
			} else {
				retChan <- retError
			}
			close(retChan)
		}()

		logTimeout := time.Second * 5
		nexTimeToLog := time.Now().Add(logTimeout)

		client, err := wgctrl.New()
		if err != nil {
			return fmt.Errorf("failed to check handshake info: %w", err)
		}
		defer client.Close()

		for ; ; time.Sleep(time.Millisecond * 50) {
			for _, isStop := range isStopArray {
				if isStop != nil && *isStop {
					return nil // stop requested (probably, disconnect requested or already disconnected)
				}
			}

			dev, err := client.Device(tunnelName)
			if err != nil {
				return fmt.Errorf("failed to check handshake info for '%s': %w", tunnelName, err)
			}

			for _, peer := range dev.Peers {
				if !peer.LastHandshakeTime.IsZero() {
					return nil // handshake detected
				}
			}

			// logging
			if logFunc != nil {
				if time.Now().After(nexTimeToLog) {
					logTimeout = logTimeout * 2
					if logTimeout > time.Second*60 {
						logTimeout = time.Second * 60
					}
					logFunc("Waiting for handshake ...")
					nexTimeToLog = time.Now().Add(logTimeout)
				}
			}
		}
	}()
	return retChan
}

func WaitForDisconnectChan(tunnelName string, isStop []*bool) <-chan error {
	return waitForWgInterfaceChan(tunnelName, true, isStop)
}
func WaitForConnectChan(tunnelName string, isStop []*bool) <-chan error {
	return waitForWgInterfaceChan(tunnelName, false, isStop)
}

// if isStopArray is defined and at lease one of it's elements == true: function stops and channel closes
func waitForWgInterfaceChan(tunnelName string, isWaitForDisconnect bool, isStopArray []*bool) <-chan error {
	retChan := make(chan error, 1)

	go func() (retError error) {
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(error); ok {
					retChan <- fmt.Errorf("crash (recovered): %w", err)
				}
			} else {
				retChan <- retError
			}
			close(retChan)
		}()

		client, err := wgctrl.New()
		if err != nil {
			return err
		}
		defer client.Close()

		for ; ; time.Sleep(time.Millisecond * 50) {
			_, err := client.Device(tunnelName)
			if isWaitForDisconnect && err != nil {
				break // waiting for Disconnect: return when error obtaining WG tunnel info
			} else if !isWaitForDisconnect && err == nil {
				break // waiting for Connect: return when NO error obtaining WG tunnel info
			}

			for _, isStop := range isStopArray {
				if isStop != nil && *isStop {
					return nil
				}
			}

		}
		return nil
	}()

	return retChan
}
