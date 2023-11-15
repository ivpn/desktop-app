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

package netchange

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ivpn/desktop-app/daemon/logger"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("NetChD")
}

// delayBeforeNotify - To avoid multiple notifications of multiple changes - the  'DelayBeforeNotify' is in use
// (notification will occur after 'DelayBeforeNotify' elapsed since last detected change)
const delayBeforeNotify = time.Second * 3

// Detector - object is detecting routing changes on a PC.
// To avoid multiple notifications of multiple changes - the  'DelayBeforeNotify' is in use
// (notification will occur after 'DelayBeforeNotify' elapsed since last detected change)
type Detector struct {
	delayBeforeNotify     time.Duration
	timerNotifyAfterDelay *time.Timer
	interfaceToProtect    *net.Interface

	isInitialised bool
	isStarted     bool
	locker        sync.Mutex

	// Signaling when the default routing is NOT over the 'interfaceToProtect' anymore
	routingChangeNotifyChan chan<- struct{}
	// Signaling when there were some routing changes but 'interfaceToProtect' is still is the default route or 'interfaceToProtect' not defined
	routingUpdateNotifyChan chan<- struct{}

	// Must be implemented (AND USED) in correspond file for concrete platform. Must contain platform-specified properties (or can be empty struct)
	props osSpecificProperties
}

// Create - create new network change detector
// 'routingChangeChan' is a notification channel
func Create() *Detector {
	// initialize detector object
	detector := &Detector{delayBeforeNotify: delayBeforeNotify}

	// initialize 'delay'-timer
	timer := time.AfterFunc(0, func() {
		detector.notifyRoutingChange()
	})
	// ensure timer is stopped (no changes detected for now)
	timer.Stop()

	// save timer
	detector.timerNotifyAfterDelay = timer

	return detector
}

// Start - start route change detector (asynchronous)
//
//	'routingUpdateChan' is the channel for notifying when there were some routing changes (but 'interfaceToProtect' is still is the default route or 'interfaceToProtect' not defined)
func (d *Detector) Init(routingChangeChan chan<- struct{}, routingUpdateChan chan<- struct{}, currentDefaultInterface *net.Interface) error {
	// Ensure that detector is stopped
	d.Stop()

	d.locker.Lock()
	defer d.locker.Unlock()

	// set notification channel (it is important to do it after we are ensure that timer is stopped)
	d.routingChangeNotifyChan = routingChangeChan
	d.routingUpdateNotifyChan = routingUpdateChan

	// save current default interface
	d.interfaceToProtect = currentDefaultInterface
	if d.interfaceToProtect == nil {
		// If 'interfaceToProtect' not defined - we do not notify to 'routingChangeChan' channel
		// only general routing chnages will be notified (using 'routingUpdateChan')
		log.Info("initialisation: 'interface to protect' not specified!")
	}

	d.isInitialised = true
	return nil
}

func (d *Detector) UnInit() error {
	d.locker.Lock()
	d.isInitialised = false
	d.locker.Unlock()

	d.Stop()

	return nil
}

func (d *Detector) Start() error {
	d.locker.Lock()
	defer d.locker.Unlock()

	if !d.isInitialised {
		return fmt.Errorf("not initialised")
	}

	if d.isStarted {
		return nil
	}
	d.isStarted = true

	// method should be implemented in platform-specific file
	go d.doStart()
	return nil
}

// Stop - stop route change detector
func (d *Detector) Stop() error {
	d.locker.Lock()
	defer d.locker.Unlock()

	d.isStarted = false
	// stop timer
	d.timerNotifyAfterDelay.Stop()
	// method should be implemented in platform-specific file
	d.doStop()
	return nil
}

// Must be called when routing change detected (called from platform-specific sources)
// It notifies about routing change with delay 'd.DelayBeforeNotify()'. This reduces amount of multiple consecutive notifications
func (d *Detector) routingChangeDetected() {
	d.timerNotifyAfterDelay.Reset(d.delayBeforeNotify)
}

// Immediately notify about routing change.
// Consider using routingChangeDetected() instead
func (d *Detector) notifyRoutingChange() {
	if d.routingChangeNotifyChan == nil {
		return
	}

	var changed bool = false
	var err error = nil

	if d.interfaceToProtect != nil {
		// Method should be implemented in platform-specific file
		// It must compare current routing configuration with configuration which was when 'doStart()' called
		if changed, err = d.isRoutingChanged(); err != nil {
			return
		}
	}

	channelToNotify := d.routingUpdateNotifyChan // there were some routing changes but 'interfaceToProtect' is still is the default route
	if changed {
		log.Info("Route change detected. Internet traffic is no longer being routed through the VPN.")
		channelToNotify = d.routingChangeNotifyChan //  the default routing is NOT over the 'interfaceToProtect' anymore
	}

	select {
	case channelToNotify <- struct{}{}:
		// notified
	default:
		// channel is full
	}
}
