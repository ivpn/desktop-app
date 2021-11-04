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

package obfsproxy

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"time"

	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/shell"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("obfpxy")
	rand.Seed(time.Now().UnixNano())
}

type startedCmd struct {
	command   *exec.Cmd
	stopped   <-chan struct{}
	exitError error
}

// Obfsproxy structure. Contains info about obfsproxy binary
type Obfsproxy struct {
	binaryPath string
	proc       *startedCmd
}

// CreateObfsproxy creates new obfsproxy object
func CreateObfsproxy(theBinaryPath string) (obj *Obfsproxy) {
	return &Obfsproxy{binaryPath: theBinaryPath}
}

// Start - asynchronously start obfsproxy
func (p *Obfsproxy) Start() (port int, err error) {
	log.Info("Starting obfsproxy")
	defer func() {
		if err != nil || port <= 0 {
			if err == nil {
				err = fmt.Errorf("error starting obfsproxy")
			}
			log.Error(err)
			p.Stop()
		}
	}()

	localPort, command, err := p.start()
	if err != nil {
		return 0, fmt.Errorf("failed to start obfsproxy: %w", err)
	}

	p.proc = command

	return localPort, nil
}

func (p *Obfsproxy) Wait() error {
	prc := p.proc
	if prc == nil {
		return nil
	}

	<-prc.stopped
	return prc.exitError
}

// Stop - stop obfsproxy
func (p *Obfsproxy) Stop() {
	prc := p.proc
	if prc == nil {
		return
	}

	log.Info("Stopping obfsproxy...")
	if err := shell.Kill(prc.command); err != nil {
		log.Error(err)
	}
}

func (p *Obfsproxy) start() (port int, command *startedCmd, err error) {
	// obfsproxy command
	cmd := exec.Command(p.binaryPath)

	ptStateDir := path.Join(platform.LogDir(), "ivpn-obfsproxy-state")
	// obfs4 configuration parameters
	// https://github.com/Pluggable-Transports/Pluggable-Transports-spec/tree/main/releases
	// https://gitweb.torproject.org/torspec.git/tree/pt-spec.txt
	const obfsProxyVer = "obfs3"
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "TOR_PT_CLIENT_TRANSPORTS="+obfsProxyVer)
	cmd.Env = append(cmd.Env, "TOR_PT_MANAGED_TRANSPORT_VER=1")
	cmd.Env = append(cmd.Env, fmt.Sprintf("TOR_PT_STATE_LOCATION=%s", ptStateDir))

	defer func() {
		if err != nil {
			// in case of error - ensure process is stopped
			shell.Kill(cmd)
			command = nil
		}
	}()

	localPort := 0
	isInitialized := false
	// output example:
	// 	VERSION 1
	// 	CMETHOD obfs3 socks5 127.0.0.1:53914
	//	CMETHODS DONE
	portRegExp := regexp.MustCompile("CMETHOD.+" + obfsProxyVer + ".+[0-9.]+:([0-9]+)")
	outputParseFunc := func(text string, isError bool) {
		if isError {
			log.Info("[ERR] ", text)
		} else {
			log.Info("[OUT] ", text)

			// check if obfsproxy ready to use
			if text == "CMETHODS DONE" {
				isInitialized = true
				return
			}

			// check for port number
			columns := portRegExp.FindStringSubmatch(text)
			if len(columns) > 1 {
				localPort, _ = strconv.Atoi(columns[1])
			}
		}
	}

	// register colsole output reader for a process
	if err := shell.StartConsoleReaders(cmd, outputParseFunc); err != nil {
		log.Error("Failed to init obfsproxy command: ", err.Error())
		return 0, nil, err
	}

	// Start obfsproxy process
	if err := cmd.Start(); err != nil {
		log.Error("Failed to start obfsproxy: ", err.Error())
		return 0, nil, err
	}

	stoppedChan := make(chan struct{}, 1)
	var procStoppedError error
	go func() {
		procStoppedError = cmd.Wait()
		log.Info("Obfsproxy stopped")
		stoppedChan <- struct{}{}
		close(stoppedChan)

		// remove PT state directory
		os.RemoveAll(ptStateDir)
	}()

	started := time.Now()
	// waiting for first channel output (ensure process is started)
	for !isInitialized && shell.IsRunning(cmd) {
		time.Sleep(time.Millisecond * 10)

		// timeout limit to start obfsproxy process = 10 seconds
		if time.Since(started) > time.Second*10 {
			return 0, nil, errors.New("obfsproxy start timeout")
		}
	}

	if localPort > 0 {
		log.Info(fmt.Sprintf("Started on port %d", localPort))
	}
	return localPort, &startedCmd{command: cmd, stopped: stoppedChan, exitError: procStoppedError}, nil
}
