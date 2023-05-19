//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 Privatus Limited.
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

type WgHandshakeTimeoutError struct {
}

func (e WgHandshakeTimeoutError) Error() string {
	return "WireGuard handshake timeout"
}

type WgDeviceNotFoundError struct {
}

func (e WgDeviceNotFoundError) Error() string {
	return "WireGuard device not found"
}

// WaitForFirstHanshake waits for a handshake during 'timeout' time.
// If no handshake occured - returns WgHandshakeTimeoutError
// If timeout == 0 - function returns only when isStop changed to true
func WaitForWireguardFirstHanshake(tunnelName string, timeout time.Duration, isStop *bool, logFunc func(string)) (retErr error) {
	if timeout == 0 && isStop == nil {
		return fmt.Errorf("internal error: bad arguments for WaitForWireguardFirstHanshake")
	}

	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				retErr = fmt.Errorf("crash (recovered): %w", err)
			}
		}
	}()

	endTime := time.Now().Add(timeout)

	logTimeout := time.Second * 5
	nexTimeToLog := time.Now().Add(logTimeout)

	client, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("failed to check handshake info: %w", err)
	}
	defer client.Close()

	for {
		if isStop != nil && *isStop {
			return WgHandshakeTimeoutError{} // disconnect requested
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

		if timeout > 0 {
			if time.Now().After(endTime) {
				return WgHandshakeTimeoutError{}
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

		// sleep before next check
		time.Sleep(time.Millisecond * 50)
	}
}
