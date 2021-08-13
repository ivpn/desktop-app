//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2021 Privatus Limited.
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

package splittun

import (
	"net"
	"sync"

	"github.com/ivpn/desktop-app/daemon/logger"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("spltun")
}

var (
	isConnected bool
	mutex       sync.Mutex
)

type State struct {
	IsConfigOk         bool
	IsEnabledSplitting bool
}

type ConfigAddresses struct {
	IPv4Public net.IP
	IPv4Tunnel net.IP
	IPv6Public net.IP
	IPv6Tunnel net.IP
}
type ConfigApps struct {
	ImagesPathToSplit []string
}

type Config struct {
	Addr ConfigAddresses
	Apps ConfigApps
}

func IsConnectted() bool {
	mutex.Lock()
	defer mutex.Unlock()

	return isConnected
}

func Initialize() error {
	mutex.Lock()
	defer mutex.Unlock()

	log.Info("Initializing Split-Tunnelling")
	err := implInitialize()
	if err != nil {
		return err
	}

	return nil
}

// IsFuncNotAvailableError - returns non-nil error object if Split-Tunneling functionality not available
func GetFuncNotAvailableError() error {
	return implFuncNotAvailableError()
}

func Connect() error {
	mutex.Lock()
	defer mutex.Unlock()

	if isConnected {
		return nil
	}

	log.Info("Split-Tunnelling: Connect...")
	ret := implConnect()
	if ret == nil {
		isConnected = true
		log.Info("Split-Tunnelling: ready")
	}
	return ret
}

func Disconnect() error {
	mutex.Lock()
	defer mutex.Unlock()

	log.Info("Split-Tunnelling: Disconnect...")
	isConnected = false
	return implDisconnect()
}

func StopAndClean() error {
	mutex.Lock()
	defer mutex.Unlock()

	log.Info("Split-Tunnelling: StopAndClean...")

	return implStopAndClean()
}

func GetState() (State, error) {
	mutex.Lock()
	defer mutex.Unlock()

	return implGetState()
}

func SetConfig(config Config) error {
	mutex.Lock()
	defer mutex.Unlock()

	log.Info("Split-Tunnelling: SetConfig...")
	return implSetConfig(config)
}
func GetConfig() (Config, error) {
	mutex.Lock()
	defer mutex.Unlock()

	return implGetConfig()
}

func Start() error {
	mutex.Lock()
	defer mutex.Unlock()

	log.Info("Split-Tunnelling: Start...")
	return implStart()
}

/*
func Stop() error {
	mutex.Lock()
	defer mutex.Unlock()

	log.Info("Split-Tunnelling: stopping")
	return implStop()
}*/
