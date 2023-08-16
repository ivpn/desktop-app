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

package dnscryptproxy

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

const configSvrName = "ivpnmanualconfig"

// SaveConfigFile - update template file 'configFileTemplate's with required data
// and save result into 'configFileOut'
// The implementation is very simple and based in replacing specific lines in template.
func SaveConfigFile(dnsSvrStamp, configFileTemplate, configFileOut string) error {
	if _, err := os.Stat(configFileTemplate); err != nil {
		return err
	}

	input, err := ioutil.ReadFile(configFileTemplate)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")

	isUpdated_server_names := false
	isUpdated_static_myserver := false
	isUpdated_stamp := false

	for i, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "# server_names = ") {
			lines[i] = fmt.Sprintf("server_names = ['%s']", configSvrName)
			isUpdated_server_names = true
		} else if strings.HasPrefix(line, "# [static.'myserver']") {
			lines[i] = fmt.Sprintf("[static.'%s']", configSvrName)
			isUpdated_static_myserver = true
		} else if strings.HasPrefix(line, "#") && strings.Contains(line, "stamp =") {
			lines[i] = fmt.Sprintf("stamp = '%s'", dnsSvrStamp)
			isUpdated_stamp = true
		}
	}

	if !isUpdated_server_names || !isUpdated_static_myserver || !isUpdated_stamp {
		return fmt.Errorf("failed to update configuration from template file")
	}

	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile(configFileOut, []byte(output), 0600) // read only for owner
	if err != nil {
		return err
	}

	return nil
}
