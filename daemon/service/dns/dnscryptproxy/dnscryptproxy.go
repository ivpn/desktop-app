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

package dnscryptproxy

import (
	"fmt"
	"sync"

	"github.com/ivpn/desktop-app/daemon/logger"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("dnscrt")
}

type iDnsCryptProxy interface {
	implStart() error
	implStop() error
}

var (
	_dnsCryptObjMutex sync.Mutex
	_dnsCryptObj      iDnsCryptProxy = nil
)

func Init(theBinaryPath, configFilePath, logFilePath string) error {
	if err := Stop(); err != nil {
		return err
	}

	_dnsCryptObjMutex.Lock()
	defer _dnsCryptObjMutex.Unlock()
	_dnsCryptObj = implInit(theBinaryPath, configFilePath, logFilePath)

	return nil
}

// Start - asynchronously start
func Start() (err error) {
	_dnsCryptObjMutex.Lock()
	defer _dnsCryptObjMutex.Unlock()

	log.Info("Starting dnscrypt-proxy")
	defer func() {
		if err != nil {
			err = fmt.Errorf("error starting dnscrypt-proxy: %w", err)
			//log.Error(err)
			_dnsCryptObj.implStop()
		}
	}()

	return _dnsCryptObj.implStart()
}

func Stop() (err error) {
	_dnsCryptObjMutex.Lock()
	defer _dnsCryptObjMutex.Unlock()

	if _dnsCryptObj == nil {
		return nil
	}

	log.Info("Stopping dnscrypt-proxy")
	defer func() {
		if err != nil {
			err = fmt.Errorf("error stopping dnscrypt-proxy: %w", err)
		}
	}()

	ret := _dnsCryptObj.implStop()
	if ret == nil {
		_dnsCryptObj = nil
	}
	return ret
}
