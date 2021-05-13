//
//  IVPN command line interface (CLI)
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
//
//  This file is part of the IVPN command line interface.
//
//  The IVPN command line interface is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The IVPN command line interface is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the IVPN command line interface. If not, see <https://www.gnu.org/licenses/>.
//

package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

func lastConnectionInfoFile() string {
	dir, err := configDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "last_connection")
}

// LastConnectionInfo information about last connection parameters
type LastConnectionInfo struct {
	Gateway         string
	Port            string
	Obfsproxy       bool
	FirewallOff     bool
	DNS             string
	Antitracker     bool
	AntitrackerHard bool

	MultiopExitSvr string // variable name spelling error ->  'MultihopExitSvr' (keeped as is for compatibility with previous versions)
}

// LastConnectionExist - returns 'true' if available info about last successful connection
func LastConnectionExist() bool {
	if _, err := os.Stat(lastConnectionInfoFile()); err == nil {
		return true
	}
	return false
}

// SaveLastConnectionInfo save last connection parameters in local storage
func SaveLastConnectionInfo(ci LastConnectionInfo) {
	data, err := json.Marshal(ci)
	if err != nil {
		return
	}

	if file := lastConnectionInfoFile(); len(file) > 0 {
		ioutil.WriteFile(file, data, 0600) // read only for owner
	}
}

// RestoreLastConnectionInfo restore last connection info from local storage
func RestoreLastConnectionInfo() *LastConnectionInfo {
	ci := LastConnectionInfo{}

	file := ""
	if file = lastConnectionInfoFile(); len(file) == 0 {
		return nil
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	err = json.Unmarshal(data, &ci)
	if err != nil {
		return nil
	}

	return &ci
}
