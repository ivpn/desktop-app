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

//go:build windows
// +build windows

package main

import (
	"fmt"

	"github.com/ivpn/desktop-app/daemon/service"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

// ServiceName -  name of the service
const _serviceName = "IVPN Client"

var _evtlog *eventlog.Log
var _stopped chan struct{}

type ivpnservice struct{}

// EventType (svc.PowerEvent) info: https://learn.microsoft.com/en-us/windows/win32/power/wm-powerbroadcast
//
//	https://learn.microsoft.com/en-us/windows/win32/api/winsvc/nc-winsvc-lphandler_function_ex
//	https://learn.microsoft.com/en-us/windows/win32/power/pbt-powersettingchange
type PowerBroadcastType uint32

const (
	PBT_APMPOWERSTATUSCHANGE PowerBroadcastType = 10    // Power status has changed.
	PBT_APMRESUMEAUTOMATIC   PowerBroadcastType = 18    // Operation is resuming automatically from a low-power state. This message is sent every time the system resumes.
	PBT_APMRESUMESUSPEND     PowerBroadcastType = 7     // Operation is resuming from a low-power state. This message is sent after PBT_APMRESUMEAUTOMATIC if the resume is triggered by user input, such as pressing a key.
	PBT_APMSUSPEND           PowerBroadcastType = 4     // System is suspending operation.
	PBT_POWERSETTINGCHANGE   PowerBroadcastType = 32787 // A power setting change event has been received.
)

type SessioChangeType uint32

const (
	WTS_SESSION_LOGON  SessioChangeType = 0x5 // A user has logged on to the session identified by lParam.
	WTS_SESSION_LOGOFF SessioChangeType = 0x6 // A user has logged off the session identified by lParam.
	// More values: https://learn.microsoft.com/en-us/windows/win32/termserv/wm-wtssession-change
)

// Prepare to start IVPN service for Windows
func doPrepareToRun() error {
	isService, err := svc.IsWindowsService()
	if err != nil {
		log.Error(fmt.Sprintf("failed to determine if we are running in as a Windows service: %v", err))
	}

	log.Info("IsWindowsService: ", isService)
	if !isService {
		log.Info("Starting as a console application (testing mode; isService=true)")
		// It is interactive session. Continue as console application (testing mode)
		return nil
	}
	log.Info("Starting as a service (InteractiveSession=false)")

	// run a service handler (service is active until 'Execute(...)' method is running)
	go runWindowsService()

	// continue starting other stuff
	return nil
}

// inform OS-specific implementation about listener port
func doStartedOnPort(port int, secret uint64) {
}

// OS-specific service finalizer
func doStopped() {
	var stoppedChan = _stopped

	if stoppedChan != nil {
		// notify service handler that service is stopped
		stoppedChan <- struct{}{}
	}
}

// doCheckIsAdmin - check is application running with root privileges
func doCheckIsAdmin() bool {
	var sid *windows.SID

	// Although this looks scary, it is directly copied from the
	// official windows documentation. The Go API for this is a
	// direct wrap around the official C++ API.
	// See https://docs.microsoft.com/en-us/windows/desktop/api/securitybaseapi/nf-securitybaseapi-checktokenmembership
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil {
		log.Error(fmt.Sprintf("SID Error: %s", err.Error()))
		return false
	}

	// https://github.com/golang/go/issues/28804#issuecomment-438838144
	token := windows.Token(0)

	// Also note that an admin is _not_ necessarily considered
	// elevated.
	// For elevation see https://github.com/mozey/run-as-admin
	//return token.IsElevated()

	member, err := token.IsMember(sid)
	if err != nil {
		log.Error(fmt.Sprintf("Token Membership Error: %s", err.Error()))
		return false
	}

	log.Info(fmt.Sprintf("IsAdmin=%v IsElevated=%v", member, token.IsElevated()))

	return member
}

func runWindowsService() {
	var err error
	_evtlog, err = eventlog.Open(_serviceName)
	if err != nil {
		log.Warning(fmt.Sprintf("Unable to initialize windows event log: %v", err))
		_evtlog = nil
	}
	defer func() {
		if _evtlog != nil {
			_evtlog.Close()
		}
	}()

	// write service messages into system log
	systemLog = make(chan service.SystemLogMessage, 1)
	go func() {
		for {
			mes := <-systemLog
			switch mes.Type {
			case service.Info:
				_evtlog.Info(0, mes.Message)
			case service.Warning:
				_evtlog.Warning(0, mes.Message)
			case service.Error:
				_evtlog.Error(0, mes.Message)
			}
		}
	}()

	log.Info(fmt.Sprintf("starting %s service", _serviceName))
	if _evtlog != nil {
		_evtlog.Info(1, fmt.Sprintf("starting %s service", _serviceName))
	}

	// create stop-detection channel
	_stopped = make(chan struct{}, 1)

	// run windows-service-handler (func (m *ivpnservice) Execute(...))
	err = svc.Run(_serviceName, &ivpnservice{})
	if err != nil {
		log.Error(fmt.Sprintf("%s service failed: %v", _serviceName, err))
		if _evtlog != nil {
			_evtlog.Error(1, fmt.Sprintf("%s service failed: %v", _serviceName, err))
		}
		return
	}

	log.Info(fmt.Sprintf("%s service stopped", _serviceName))
	if _evtlog != nil {
		_evtlog.Info(1, fmt.Sprintf("%s service stopped", _serviceName))
	}
}

func (m *ivpnservice) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	log.Info("Service handler started")
	defer func() {
		changes <- svc.Status{State: svc.StopPending}
		log.Info("Service handler: StopPending")

		// Stop the service (if not stopped yet)
		// This call should be performed at the end. Application will fully stop after that
		Stop()

		changes <- svc.Status{State: svc.Stopped}
		log.Info("Service handler: Stopped")
	}()

	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPowerEvent | svc.AcceptSessionChange
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

loop:
	for {
		select {
		case <-_stopped:
			log.Info("Service stopped")
			break loop

		case c := <-r:
			switch c.Cmd {

			case svc.Interrogate:
				// SERVICE_CONTROL_INTERROGATE
				// 0x00000004
				// Notifies a service that it should report its current status information to the service control manager. The hService handle must have the SERVICE_INTERROGATE access right.
				// Note that this control is not generally useful as the SCM is aware of the current state of the service.

				log.Info("Service control request: ", "Interrogate", c.Cmd)

				changes <- c.CurrentStatus

				// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				//time.Sleep(100 * time.Millisecond)
				//changes <- c.CurrentStatus

			case svc.Stop, svc.Shutdown:
				log.Info("Service control request: ", "Stop|Shutdown", c.Cmd)
				if _evtlog != nil {
					_evtlog.Info(1, fmt.Sprintf("Service control request: Stop|Shutdown %d", c.Cmd))
				}
				break loop

			case svc.PowerEvent:
				if c.EventType == uint32(PBT_APMRESUMEAUTOMATIC) {
					serviceEventNotify(service.On_Power_WakeUp)
				}

			case svc.SessionChange:
				if c.EventType == uint32(WTS_SESSION_LOGON) {
					serviceEventNotify(service.On_Session_Logon)
				}
			default:
				log.Warning("Unexpected service control request: ", c.Cmd)
				if _evtlog != nil {
					_evtlog.Error(1, fmt.Sprintf("unexpected control request #%d", c))
				}
			}
		}
	}

	return
}

func isNeedToSavePortInFile() bool {
	return true
}
