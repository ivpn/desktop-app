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

//go:build linux
// +build linux

package dnscryptproxy

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/shell"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("dnscrt")
}

type startedCmd struct {
	command   *exec.Cmd
	stopped   <-chan struct{}
	exitError error
}

type DnsCryptProxy struct {
	binaryPath     string
	configFilePath string
	proc           *startedCmd
}

func CreateDnsCryptProxy(theBinaryPath string, configFilePath string) (obj *DnsCryptProxy) {
	return &DnsCryptProxy{binaryPath: theBinaryPath, configFilePath: configFilePath}
}

// Start - asynchronously start
func (p *DnsCryptProxy) Start() (err error) {
	log.Info("Starting dnscrypt-proxy")
	defer func() {
		if err != nil {
			if err == nil {
				err = fmt.Errorf("error starting dnscrypt-proxy")
			}
			log.Error(err)
			p.Stop()
		}
	}()

	command, err := p.start()
	if err != nil {
		return err
	}

	p.proc = command

	return nil
}

func (p *DnsCryptProxy) Wait() error {
	prc := p.proc
	if prc == nil {
		return nil
	}

	<-prc.stopped
	return prc.exitError
}

func (p *DnsCryptProxy) Stop() {
	prc := p.proc
	if prc == nil {
		return
	}

	log.Info("Stopping dnscrypt-proxy...")
	if err := shell.Kill(prc.command); err != nil {
		log.Error(err)
	}
}

func (p *DnsCryptProxy) start() (command *startedCmd, err error) {
	//cmd := exec.Command(p.binaryPath, "-child", "-config", p.configFilePath)
	cmd := exec.Command(p.binaryPath, "-config", p.configFilePath)
	log.Debug(cmd)

	defer func() {
		if err != nil {
			// in case of error - ensure process is stopped
			shell.Kill(cmd)
			command = nil
		}
	}()

	var lastOutError error = nil
	isInitialized := false
	// output example:
	// 	[NOTICE] [ivpnmanualconfig] OK
	outputParseFunc := func(text string, isError bool) {
		log.Info("[OUT] ", text)
		// check if dnscrypt-proxy ready to use
		if strings.Contains(text, "[NOTICE] Now listening to") {
			isInitialized = true
			return
		}
		if strings.Contains(text, " [FATAL] ") {
			lastOutError = fmt.Errorf(text)
		}
	}

	// register colsole output reader for a process
	if err := shell.StartConsoleReaders(cmd, outputParseFunc); err != nil {
		log.Error("Failed to init dnscrypt-proxy command: ", err.Error())
		return nil, err
	}

	// Start process
	if err := cmd.Start(); err != nil {
		log.Error("Failed to start dnscrypt-proxy: ", err.Error())
		return nil, err
	}

	stoppedChan := make(chan struct{}, 1)
	var procStoppedError error
	go func() {
		procStoppedError = cmd.Wait()
		log.Info("dnscrypt-proxy stopped")
		stoppedChan <- struct{}{}
		close(stoppedChan)
	}()

	started := time.Now()
	// waiting for first channel output (ensure process is started)
	for !isInitialized {
		if !shell.IsRunning(cmd) {
			var exitCode int = 0
			procStoppedError = cmd.Wait()
			if procStoppedError != nil {
				exitCode, _ = shell.GetCmdExitCode(procStoppedError)
			}

			if lastOutError != nil {
				return nil, fmt.Errorf("%w (retcode=%d)", lastOutError, exitCode)
			} else {
				return nil, fmt.Errorf("dnscrypt-proxy error (retcode=%d)", exitCode)
			}
		}

		time.Sleep(time.Millisecond * 10)

		// timeout limit to start dnscrypt-proxy process = 10 seconds
		if time.Since(started) > time.Second*20 {
			return nil, errors.New("dnscrypt-proxy start timeout")
		}
	}

	log.Info("dnscrypt-proxy started")
	return &startedCmd{command: cmd, stopped: stoppedChan, exitError: procStoppedError}, nil
}
