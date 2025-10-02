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

package dnscryptproxy

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ivpn/desktop-app/daemon/shell"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const _WIN_SERVICE_NAME = "dnscrypt-proxy"

type extraParams struct {
}

// Start - asynchronously start
func (p *DnsCryptProxy) implStart() (retErr error) {
	defer func() {
		if retErr != nil {
			log.Error(retErr)
			shell.Exec(nil, p.binaryPath, "-service", "uninstall")
		}
	}()

	p.implStop()

	if len(p.logFilePath) > 0 {
		os.Remove(p.logFilePath)
	}

	log.Info("Installing dnscrypt-proxy service...")
	err := shell.Exec(nil, p.binaryPath, "-service", "install", "-config", p.configFilePath)
	if err != nil {
		return fmt.Errorf("failed to install service: %w", err)
	}

	log.Info("Starting dnscrypt-proxy service...")
	err = shell.Exec(nil, p.binaryPath, "-service", "start")
	if err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	// NOTE! small delay: let chance service to initialize (and to save error data into log file)
	// TODO: avoid using delays
	time.Sleep(time.Millisecond * 200)

	// check if service started
	isInstalled, isRunning, err := p.checkIsServiceRunning()
	if err != nil {
		return fmt.Errorf("failed to check status of dnscrypt-proxy service: %w", err)
	}
	if !isInstalled || !isRunning {
		// read fatal errors from log
		errText, err := p.getFatalErrorFromLog()
		if err == nil && len(errText) > 0 {
			return fmt.Errorf("dnscrypt-proxy service not started: %q", errText)
		}
		return fmt.Errorf("dnscrypt-proxy service not started")
	}
	// read fatal errors from log
	errText, err := p.getFatalErrorFromLog()
	if err == nil && len(errText) > 0 {
		return fmt.Errorf("%s", errText)
	}
	return nil
}

func (p *DnsCryptProxy) implStop() error {

	isInstalled, isRunning, statusErr := p.checkIsServiceRunning()
	if !isInstalled && !isRunning && statusErr == nil {
		// nothing to stop
		return nil
	}

	var reterr error
	if statusErr != nil || isRunning {
		log.Info("Stopping dnscrypt-proxy service...")
		err := shell.Exec(nil, p.binaryPath, "-service", "stop")
		if err != nil {
			reterr = fmt.Errorf("failed to stop service: %w", err)
		}
	}

	if statusErr != nil || isInstalled {
		log.Info("Uninstalling dnscrypt-proxy service...")
		err := shell.Exec(nil, p.binaryPath, "-service", "uninstall")
		if err != nil {
			reterr = fmt.Errorf("failed to uninstall service: %w", err)
		}
	}
	return reterr
}

func (p *DnsCryptProxy) checkIsServiceRunning() (isInstalled bool, isRunning bool, retErr error) {
	// connect to service maneger
	m, err := mgr.Connect()
	if err != nil {
		return false, false, fmt.Errorf("failed to connect windows service manager: %w", err)
	}
	defer m.Disconnect()

	// looking for service
	s, err := m.OpenService(_WIN_SERVICE_NAME)
	if err != nil {
		return false, false, nil // service not available
	}
	defer s.Close()

	// requesting service status
	status, _ := s.Query()

	switch status.State {
	case svc.Running, svc.StartPending, svc.ContinuePending:
		return true, true, nil
	}

	return true, false, nil
}

func (p *DnsCryptProxy) getFatalErrorFromLog() (string, error) {
	if _, err := os.Stat(p.logFilePath); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("stat log file: %w", err)
	}

	file, err := os.Open(p.logFilePath)
	if err != nil {
		log.Debug(fmt.Sprintf("From log %q: %q", p.logFilePath, err.Error()))
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 256)
	scanner.Buffer(buf, 1024) // limit line size to 1024 bytes

	// limit number of lines to read (500 lines max)
	lineCount := 0
	for scanner.Scan() && lineCount < 500 {
		text := scanner.Text()
		if strings.Contains(text, " [FATAL] ") {
			return text, nil
		}
		lineCount++
	}

	return "", nil
}
