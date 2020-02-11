package main

import (
	"fmt"
	"ivpn/daemon/logger"
	"ivpn/daemon/netchange"
	"ivpn/daemon/protocol"
	"ivpn/daemon/service"
	"ivpn/daemon/service/api"
	"ivpn/daemon/service/platform"
	"os"
	"runtime"
	"strconv"
	"time"
)

var log *logger.Logger
var activeProtocol service.Protocol

func init() {
	log = logger.NewLogger("launch")
}

// Launch -  initialize and start service
func Launch() {
	defer func() {
		log.Info("IVPN daemon stopped.")

		// OS-specific service finalizer
		doStopped()
	}()

	tzName, tzOffsetSec := time.Now().Zone()
	log.Info("Starting IVPN daemon", fmt.Sprintf(" [%s]", runtime.GOOS), fmt.Sprintf(" [timezone: %s %d (%dh)]", tzName, tzOffsetSec, tzOffsetSec/(60*60)), " ...")
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

	// obtain (over callback channel) a service listening port
	startedOnPortChan := make(chan int, 1)
	go func() {
		// waiting for port number info
		openedPort := <-startedOnPortChan

		// for Windows and macOS-debug we need to save port info into a file
		if isNeedToSavePortInFile() == true {
			file, err := os.Create(platform.ServicePortFile())
			if err != nil {
				logger.Panic(err.Error())
			}
			defer file.Close()
			file.WriteString(fmt.Sprintf("%d", openedPort))
		}
		// inform OS-specific implementation about listener port
		doStartedOnPort(openedPort)
	}()

	defer func() {
		if isNeedToSavePortInFile() == true {
			os.Remove(platform.ServicePortFile())
		}
	}()

	// perform OS-specific preparetions (if necessary)
	if err := doPrepareToRun(); err != nil {
		logger.Panic(err.Error())
	}

	// run service
	launchService(startedOnPortChan)
}

// Stop the service
func Stop() {
	p := activeProtocol
	if p != nil {
		p.Stop()
	}
}

// initialize and start service
func launchService(startedOnPort chan<- int) {
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

	// initialize service
	serv, err := service.CreateService(updater, netDetector)
	if err != nil {
		log.Panic("Failed to initialize service:", err)
	}

	// communication protocol
	protocol, err := protocol.CreateProtocol()
	if err != nil {
		log.Panic("Protocol object initialization failed: ", err)
	}

	// save protocol (to be able to stop it)
	activeProtocol = protocol

	// start receiving requests from client (synchronous)
	if err := protocol.Start(startedOnPort, serv); err != nil {
		log.Error("Protocol stopped with error:", err)
	}
}
