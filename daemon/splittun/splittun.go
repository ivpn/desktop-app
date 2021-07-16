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

	"github.com/ivpn/desktop-app/daemon/logger"
)

var log *logger.Logger
var isConnected bool

func init() {
	log = logger.NewLogger("spltun")
}

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

func Initialize() error {
	err := implInitialize()
	if err != nil {
		return err
	}

	err = Connect()
	if err != nil {
		return err
	}

	return nil
}

func Connect() error {
	if isConnected {
		return nil
	}
	ret := implConnect()
	if ret == nil {
		isConnected = true
	}
	return ret
}
func Disconnect() error {
	return implDisconnect()
}

func StopAndClean() error {
	return implStopAndClean()
}

func GetState() (State, error) {
	return implGetState()
}

func SetConfig(config Config) error {
	return implSetConfig(config)
}
func GetConfig() (Config, error) {
	return implGetConfig()
}

func Start() error {
	return implStart()
}
func Stop() error {
	return implStop()
}
