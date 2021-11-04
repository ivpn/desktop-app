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

package main

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ivpn/desktop-app/daemon/api"
	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/netchange"
	"github.com/ivpn/desktop-app/daemon/protocol"
	"github.com/ivpn/desktop-app/daemon/service"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/service/preferences"
	"github.com/ivpn/desktop-app/daemon/service/wgkeys"
	"github.com/ivpn/desktop-app/daemon/version"
)

var log *logger.Logger
var activeProtocol IProtocol

func init() {
	log = logger.NewLogger("launch")
	rand.Seed(time.Now().UnixNano())
}

// IProtocol - interface of communication protocol with IVPN application
type IProtocol interface {
	Start(secret uint64, startedOnPort chan<- int, serv protocol.Service) error
	Stop()
}

// Launch -  initialize and start service
func Launch() {
	defer func() {
		log.Info("IVPN daemon stopped.")

		// OS-specific service finalizer
		doStopped()
	}()

	warnings, errors := platform.Init()
	logger.Init(platform.LogFile())

	// Enable logging (if necessary)
	// Logging can be enabled from command lone (-logging)
	// or from previously saved daemon preferences
	isLoggingEnabledArgument := false
	for _, arg := range os.Args {
		arg = strings.ToLower(arg)
		if arg == "-logging" || arg == "--logging" {
			isLoggingEnabledArgument = true
			break
		}
	}
	if isLoggingEnabledArgument {
		logger.Enable(true)
		logger.Info("Loggin enabled (forced by command line argument)")
	} else {
		// initialize logging according to service preferences
		var prefs preferences.Preferences
		if err := prefs.LoadPreferences(); err == nil {
			logger.Enable(prefs.IsLogging)
		}
	}

	logger.Info("version:" + version.GetFullVersion())

	if len(warnings) > 0 {
		for _, w := range warnings {
			logger.Warning(w)
		}
	}

	if len(errors) > 0 {
		for _, e := range errors {
			logger.Error(e)
		}

		logger.Info("Daemon failed to start due to initialization errors")
		os.Exit(1)
		return
	}

	tzName, tzOffsetSec := time.Now().Zone()
	log.Info("Starting IVPN daemon", fmt.Sprintf(" [%s,%s]", runtime.GOOS, runtime.GOARCH), fmt.Sprintf(" [timezone: %s %d (%dh)]", tzName, tzOffsetSec, tzOffsetSec/(60*60)), " ...")
	log.Info(fmt.Sprintf("args: %s", os.Args))
	log.Info(fmt.Sprintf("pid : %d ppid: %d", os.Getpid(), os.Getppid()))
	log.Info(fmt.Sprintf("arch: %d bit", strconv.IntSize))

	if !doCheckIsAdmin() {
		logger.Warning("------------------------------------")
		logger.Warning("!!! NOT A PRIVILEGED USER !!!")
		logger.Warning("Please, ensure you are running an application with privileged rights.")
		logger.Warning("Otherwise, application will not work correctly.")
		logger.Warning("------------------------------------")
	}

	secret := rand.Uint64()

	// obtain (over callback channel) a service listening port
	startedOnPortChan := make(chan int, 1)
	go func() {
		// waiting for port number info
		openedPort := <-startedOnPortChan

		// save port info into a file (UI clients is able to read it)
		if isNeedToSavePortInFile() == true {
			file, err := os.Create(platform.ServicePortFile())
			if err != nil {
				logger.Panic(err.Error())
			}
			defer file.Close()
			file.WriteString(fmt.Sprintf("%d:%x", openedPort, secret))
		}
		// inform OS-specific implementation about listener port
		doStartedOnPort(openedPort, secret)
	}()

	defer func() {
		if isNeedToSavePortInFile() == true {
			os.Remove(platform.ServicePortFile())
		}
	}()

	// perform OS-specific preparations (if necessary)
	if err := doPrepareToRun(); err != nil {
		logger.Panic(err.Error())
	}

	// run service
	launchService(secret, startedOnPortChan)
}

// Stop the service
func Stop() {
	p := activeProtocol
	if p != nil {
		p.Stop()
	}
}

// initialize and start service
func launchService(secret uint64, startedOnPort chan<- int) {
	// API object
	apiObj, err := api.CreateAPI()
	if err != nil {
		log.Panic("API object initialization failed: ", err)
	}

	// servers updater
	updater, err := service.CreateServersUpdater(apiObj)
	if err != nil {
		log.Panic("ServersUpdater initialization failed: ", err)
	}

	// network change detector
	netDetector := netchange.Create()

	// WireGuard keys manager
	wgKeysMgr := wgkeys.CreateKeysManager(apiObj, platform.WgToolBinaryPath())

	// communication protocol
	protocol, err := protocol.CreateProtocol()
	if err != nil {
		log.Panic("Protocol object initialization failed: ", err)
	}

	// save protocol (to be able to stop it)
	activeProtocol = protocol

	// initialize service
	serv, err := service.CreateService(protocol, apiObj, updater, netDetector, wgKeysMgr)
	if err != nil {
		log.Panic("Failed to initialize service:", err)
	}

	// start receiving requests from client (synchronous)
	if err := protocol.Start(secret, startedOnPort, serv); err != nil {
		log.Error("Protocol stopped with error:", err)
	}
}
