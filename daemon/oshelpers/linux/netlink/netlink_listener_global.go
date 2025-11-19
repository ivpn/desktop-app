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

//go:build linux
// +build linux

package netlink

import (
	"fmt"
	"sync"
	"syscall"

	"github.com/ivpn/desktop-app/daemon/logger"
)

var (
	mutex              sync.RWMutex
	globalListener     *Listener
	globalEvtReceivers []chan NetChangeEvt
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("netlnk")
}

type NetChangeEvt struct{}

func RegisterLanChangeListener() (<-chan NetChangeEvt, error) {
	mutex.Lock()
	defer mutex.Unlock()

	receiver := make(chan NetChangeEvt, 1)

	var err error
	if globalListener == nil {
		globalListener, err = CreateListener()
		if err != nil {
			return nil, fmt.Errorf("(LAN change monitor) Netlink listener initialization error: %w", err)
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error(r)
				}

				// Cleanup on exit
				mutex.Lock()
				if globalListener != nil {
					globalListener.Close()
					globalListener = nil
				}
				for _, c := range globalEvtReceivers {
					close(c)
				}
				globalEvtReceivers = nil
				mutex.Unlock()

			}()

			log.Info("LAN change monitor started")
			defer log.Info("LAN change monitor stopped")

			for {
				msgs, err := globalListener.ReadMsgs()
				if err != nil {
					log.Error(fmt.Errorf("error parsing netlink message (LAN change monitor stopped): %w", err))
					break
				}

				isChanged := false
				for i := range msgs {
					m := msgs[i]
					if m.Header.Type == syscall.RTM_NEWADDR ||
						m.Header.Type == syscall.RTM_DELADDR ||
						m.Header.Type == syscall.RTM_NEWROUTE ||
						m.Header.Type == syscall.RTM_DELROUTE {
						isChanged = true
						break
					}
				}

				if isChanged {
					func() { // using anonymous function to unlock mutex correctly
						mutex.RLock()
						defer mutex.RUnlock()

						// notify all receivers about network change
						for _, c := range globalEvtReceivers {
							select {
							case c <- struct{}{}: // notified
							default: // channel is full
							}
						}
					}()
				}
			}
		}()
	}

	globalEvtReceivers = append(globalEvtReceivers, receiver)
	log.Info("New listener registered")

	return receiver, nil
}

func UnregisterLanChangeListener(receiver <-chan NetChangeEvt) error {
	if receiver == nil {
		return nil
	}

	mutex.Lock()
	defer mutex.Unlock()

	// Find and remove the channel from globalEvtReceivers while preserving order
	for i, c := range globalEvtReceivers {
		if c == receiver {
			// Remove by shifting elements to preserve order
			globalEvtReceivers = append(globalEvtReceivers[:i], globalEvtReceivers[i+1:]...)
			close(c)
			log.Info("Listener unregistered")
			break
		}
	}

	// If no more receivers, stop the global listener
	if len(globalEvtReceivers) == 0 && globalListener != nil {
		if err := globalListener.Close(); err != nil {
			log.Error(fmt.Errorf("error closing netlink listener: %w", err))
		}
		globalListener = nil
		log.Info("LAN change monitor cleanup completed")
	}

	return nil
}
