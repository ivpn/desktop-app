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
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
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

// Device retrieves a WireGuard device by its interface name.
//
// Purpose:
//
//	It similar to original "wgctrl.Client.Device(name string)" function.
//	But the original function has bug (in Windows implementation):
//	it hangs forever when the device does not exist anymore (https://github.com/WireGuard/wgctrl-go/issues/134)
//	Note: This function reduces chance to get deadlock but it still can happen.
func GetCtrlDevice(devName string, client *wgctrl.Client) (retDev *wgtypes.Device, err error) {
	if client == nil {
		client, err = wgctrl.New()
		if err != nil {
			return nil, err
		}
		defer client.Close()
	}

	var devs []*wgtypes.Device
	var devsErr error

	done := make(chan struct{})
	go func() {
		defer close(done)
		// Rrefresh devices.
		// TODO: potential deadlock here!
		devs, devsErr = client.Devices()
		// macOS:	there is a chance to get error "connection refused" if there another WG connection available
		//			If so, we can try to get device info directly
		if devsErr != nil {
			retDev, devsErr = client.Device(devName)
		}
	}()

	select {
	case <-done: //  client.Devices() finished
		if devsErr != nil {
			return nil, devsErr
		}
	case <-time.After(time.Second * 5): // deadlock detection timeout - 5 seconds
		log.Error("internal error: wgctrl hung")
		return nil, fmt.Errorf("internal error: wgctrl hung")
	}

	if retDev == nil {
		for _, d := range devs {
			if d.Name == devName {
				retDev = d
				break
			}
		}
	}

	if retDev == nil {
		return nil, WgDeviceNotFoundError{}
	}
	return retDev, nil
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

		dev, err := GetCtrlDevice(tunnelName, client)
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
