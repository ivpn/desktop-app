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

	"github.com/ivpn/desktop-app/daemon/logger"
)

var (
	mutex              sync.RWMutex
	globalListener     *Listener
	globalEvtReceivers []chan<- struct{}
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("netlnk")
}

func RegisterLanChangeListener(onChange chan<- struct{}) error {
	if onChange == nil {
		return nil
	}

	mutex.Lock()
	defer mutex.Unlock()

	var err error
	if globalListener == nil {
		globalListener, err = CreateListener()
		if err != nil {
			return fmt.Errorf("(LAN change monitor) Netlink listener initialization error: %w", err)
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error(r)
				}
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
					if IsNewAddr(&m) || IsDelAddr(&m) {
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

	globalEvtReceivers = append(globalEvtReceivers, onChange)
	log.Info("New listener registered")

	return nil
}
