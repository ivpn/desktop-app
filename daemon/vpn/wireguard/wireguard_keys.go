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

package wireguard

import (
	"io"
	"os/exec"
	"strings"
)

// GenerateKeys generates new WireGuard keys pair
func GenerateKeys(wgToolBinaryPath string) (publicKey string, privateKey string, err error) {
	// private key
	privCmd := exec.Command(wgToolBinaryPath, "genkey")
	out, err1 := privCmd.Output()
	if err1 != nil {
		return "", "", err1
	}
	privateKey = string(out)

	// public key
	pubCmd := exec.Command(wgToolBinaryPath, "pubkey")
	stdin, err2 := pubCmd.StdinPipe()
	if err2 != nil {
		return "", "", err2
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, privateKey) // write to command stdin
	}()

	out, err = pubCmd.Output()
	if err != nil {
		return "", "", err
	}
	publicKey = string(out)

	return strings.TrimSpace(publicKey), strings.TrimSpace(privateKey), nil
}
