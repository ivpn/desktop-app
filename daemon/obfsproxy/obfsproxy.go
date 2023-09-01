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

package obfsproxy

import (
	"errors"
	"fmt"
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
}

type ObfsProxyVersion int

const (
	None  ObfsProxyVersion = 0
	OBFS3 ObfsProxyVersion = 3
	OBFS4 ObfsProxyVersion = 4
)

// Obfs4IatMode - Inter-Arrival Time (IAT)
//
//	The values of IAT-mode can be “0”, “1”, or “2” in obfs4
//	0 -	means that the IAT-mode is disabled and that large packets will be split by the network drivers,
//		whose network fingerprints could be detected by censors.
//	1 - means splitting large packets into MTU-size packets instead of letting the network drivers do it.
//		Here, the MTU is 1448 bytes for the Obfs4 Bridge. This means the smaller packets cannot be reassembled for analysis and censoring.
//	2 - means splitting large packets into variable size packets. The sizes are defined in Obfs4.
type Obfs4IatMode int

const (
	Obfs4IatOff        Obfs4IatMode = 0
	Obfs4IatOn         Obfs4IatMode = 1
	Obfs4IatOnParanoid Obfs4IatMode = 2
)

type Config struct {
	Version  ObfsProxyVersion
	Obfs4Iat Obfs4IatMode
}

// IsObfsproxy returns 'true' when enabled
func (c Config) IsObfsproxy() bool {
	switch c.Version {
	case OBFS3, OBFS4:
	default:
		return false
	}
	return true
}

func (c Config) Equals(b Config) bool {
	if c.IsObfsproxy() != b.IsObfsproxy() {
		return false
	}
	if c.Version == b.Version && c.Version == OBFS3 {
		return true
	}
	return c.Version == b.Version && c.Obfs4Iat == b.Obfs4Iat
}

func (c Config) ToString() string {
	if !c.IsObfsproxy() {
		return "disabled"
	}
	if c.Version == OBFS4 {
		return fmt.Sprintf("obfs%d, IAT%d", c.Version, c.Obfs4Iat)
	}
	return fmt.Sprintf("obfs%d", c.Version)
}

type startedCmd struct {
	command   *exec.Cmd
	stopped   <-chan struct{}
	exitError error
}

// Obfsproxy structure. Contains info about obfsproxy binary
type Obfsproxy struct {
	binaryPath string
	config     Config
	proc       *startedCmd
}

// CreateObfsproxy creates new obfsproxy object
func CreateObfsproxy(theBinaryPath string, conf Config) (obj *Obfsproxy) {
	return &Obfsproxy{binaryPath: theBinaryPath, config: conf}
}

func (p *Obfsproxy) MakeObfs4AuthFileContent(cert string) string {
	if p.config.Version != OBFS4 {
		return ""
	}
	// obfs4 authentication file format:
	//	cert=<server certificate>;
	//	iat-mode=0
	return fmt.Sprintf("cert=%s;\niat-mode=%d", cert, p.config.Obfs4Iat)
}

func (p *Obfsproxy) Config() Config {
	return p.config
}

// Start - asynchronously start obfsproxy
func (p *Obfsproxy) Start() (port int, err error) {
	log.Info(fmt.Sprintf("Starting obfsproxy [%s]", p.config.ToString()))
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

	ptStateDir := path.Join(platform.LogDir(), "ivpn-obfsproxy-state")

	// obfsproxy command
	cmd := exec.Command(p.binaryPath)

	// obfs4 configuration parameters
	// https://github.com/Pluggable-Transports/Pluggable-Transports-spec/tree/main/releases
	// https://gitweb.torproject.org/torspec.git/tree/pt-spec.txt
	// https://www.fortinet.com/blog/threat-research/dissecting-tor-bridges-pluggable-transport-part-2

	obfsProxyVer := "obfs4"
	if p.config.Version == OBFS3 {
		obfsProxyVer = "obfs3"
	}
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
