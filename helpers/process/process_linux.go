//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app-daemon
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

// +build linux

package process

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ivpn/desktop-app-daemon/shell"
)

// doGetPortOwnerPID returns PID of a process which is an owning of local TCP port
func doGetPortOwnerPID(localTCPPort int) (int, error) {
	//  lsof -i tcp:52994
	outText, _, exitCode, err := shell.ExecAndGetOutput(nil, 2048, "", "lsof", "-i", fmt.Sprintf("tcp:%d", localTCPPort))
	if err != nil {
		return -1, fmt.Errorf("Unable to determine PID of port owner for TCP:%d", localTCPPort)
	}
	if exitCode != 0 {
		return -1, fmt.Errorf("Unable to determine PID of port owner for TCP:%d [exit code: %d]", localTCPPort, exitCode)
	}

	// Output example (macOS):
	// 		>> sudo lsof -i tcp:52940
	// 		COMMAND     PID   USER   FD   TYPE             DEVICE SIZE/OFF NODE NAME
	// 		IVPN\x20A 20716   root   10u  IPv4 0xf2314f7fd170d1e5      0t0  TCP localhost:64245->localhost:52940 (ESTABLISHED)
	// 		ivpn-ui   30561 user   32u  IPv4 0xf2314f7fd05c7805      0t0  TCP localhost:52940->localhost:64245 (ESTABLISHED)
	// Output example (Ubuntu Linux):
	//		>> sudo lsof -i tcp:52994
	// 		COMMAND  PID   USER   FD   TYPE DEVICE SIZE/OFF NODE NAME
	// 		ivpn-ui 2353 user   46u  IPv4  41298      0t0  TCP localhost:52994->localhost:41587 (ESTABLISHED)

	regextStr := fmt.Sprintf("^[ \\t]*[^ \\t]+[ \\t]+([0-9]+)[ \\t]+.*TCP[ \\t]+localhost:%d->localhost:[0-9]+.*\\(ESTABLISHED\\).*", localTCPPort)
	rexp := regexp.MustCompile(regextStr)
	lines := strings.Split(outText, "\n")
	for _, line := range lines {
		submaches := rexp.FindStringSubmatch(line)
		if len(submaches) >= 2 {
			return strconv.Atoi(submaches[1])
		}
	}
	return -1, fmt.Errorf("Port owner PID for TCP:%d not found", localTCPPort)
}

// doGetBinaryPathByPID returns absolute path of process binary
func doGetBinaryPathByPID(pid int) (string, error) {
	// TODO! WARNING: 'ps' command does not return fill path if binary was started by symlink in located in %PATH%
	// ps 72045
	outText, _, exitCode, err := shell.ExecAndGetOutput(nil, 2048, "", "ps", fmt.Sprintf("%d", pid))
	if err != nil {
		return "", fmt.Errorf("Unable to determine binary path of PID:%d", pid)
	}
	if exitCode != 0 {
		return "", fmt.Errorf("Unable to determine binary path of PID:%d [exit code: %d]", pid, exitCode)
	}

	// Output example (macOS):
	// 		>> ps 72045
	// 		PID   TT  STAT      TIME COMMAND
	// 		72045   ??  S      0:04.69 /Applications/IVPN.app/Contents/MacOS/ivpn-ui
	// Output example (Ubuntu Linux):
	//		>> ps 2353
	//		PID TTY      STAT   TIME COMMAND
	//		2353 ?        Sl     0:02 /opt/ivpn/ui/bin/ivpn-ui

	regextStr := fmt.Sprintf("^[ \\t]*%d[ \\t]+.+[ \\t]+(.+)$", pid)
	rexp := regexp.MustCompile(regextStr)
	lines := strings.Split(outText, "\n")
	for _, line := range lines {
		submaches := rexp.FindStringSubmatch(line)
		if len(submaches) >= 2 && len(submaches[1]) > 0 {
			return submaches[1], nil
		}
	}
	return "", fmt.Errorf("Binary path of PID:%d not found", pid)
}
