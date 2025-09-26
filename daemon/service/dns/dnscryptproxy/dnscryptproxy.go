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

	"github.com/ivpn/desktop-app/daemon/logger"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("dnscrt")
}

type DnsCryptProxy struct {
	binaryPath     string
	configFilePath string
	logFilePath    string      // optional
	extra          extraParams // platform-specific extra params
}

// Start - create and start dnscrypt-proxy instance asynchronously
func Start(binaryPath, configFile, logFilePath string) (obj *DnsCryptProxy, retErr error) {
	p := &DnsCryptProxy{
		binaryPath:     binaryPath,
		configFilePath: configFile,
		logFilePath:    logFilePath,
	}

	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("error starting dnscrypt-proxy: %w", retErr)
			p.implStop()
		}
	}()

	return p, p.implStart()
}
func (p *DnsCryptProxy) Stop() (err error) {
	if p == nil {
		return nil
	}

	log.Info("Stopping dnscrypt-proxy")
	defer func() {
		if err != nil {
			err = fmt.Errorf("error stopping dnscrypt-proxy: %w", err)
		}
	}()

	ret := p.implStop()
	if ret == nil {
		p = nil
	}
	return ret
}
