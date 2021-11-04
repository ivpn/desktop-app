//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
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

package obfsproxy_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/obfsproxy"
	"github.com/ivpn/desktop-app/daemon/service/platform"
)

func TestStart(t *testing.T) {
	platform.Init()
	logger.Enable(true)
	obfsp := obfsproxy.CreateObfsproxy(platform.ObfsproxyStartScript())

	port, err := obfsp.Start()
	if err != nil {
		fmt.Println("ERROR:", err)
	} else {
		fmt.Println("Started on:", port)
	}

	go func() {
		time.Sleep(time.Second * 5)
		obfsp.Stop()
	}()

	if err := obfsp.Wait(); err != nil {
		fmt.Println("STOP ERROR:", err)
	}
	fmt.Println("STOPED")
}
