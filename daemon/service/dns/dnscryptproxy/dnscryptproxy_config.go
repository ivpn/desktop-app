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
	"os"
	"strings"
)

const configSvrName = "ivpnmanualconfig"

// SaveConfigFile - update template file 'configFileTemplate's with required data
// and save result into 'configFileOut'
// The implementation is very simple and based in replacing specific lines in template.
func SaveConfigFile(dnsSvrStamps []string, configFileTemplate, configFileOut, logFilePath string) error {
	if _, err := os.Stat(configFileTemplate); err != nil {
		return err
	}

	input, err := os.ReadFile(configFileTemplate)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")
	out := strings.Builder{}

	isUpdated_server_names := false
	isUpdated_static_myserver := false
	isUpdated_stamp := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "# server_names = ") {
			server_names := make([]string, 0, len(dnsSvrStamps))
			for i := 1; i <= len(dnsSvrStamps); i++ {
				server_names = append(server_names, fmt.Sprintf("'%s%d'", configSvrName, i))
			}
			out.WriteString(fmt.Sprintf("server_names = [%s]\n", strings.Join(server_names, ", ")))
			isUpdated_server_names = true

		} else if strings.HasPrefix(line, "# [static.myserver]") {
			for j, stamp := range dnsSvrStamps {
				out.WriteString(fmt.Sprintf("\n[static.%s%d]\n", configSvrName, j+1))
				out.WriteString(fmt.Sprintf("stamp = '%s'\n", stamp))
			}
			isUpdated_static_myserver = true
			isUpdated_stamp = true

		} else if strings.HasPrefix(line, "#") && strings.Contains(line, "stamp =") && isUpdated_static_myserver {
			continue

		} else if strings.HasPrefix(line, "# log_file =") && len(logFilePath) > 0 {
			out.WriteString(fmt.Sprintf("log_file = '%s'\n", logFilePath))

		} else {
			out.WriteString(line + "\n")
		}
	}

	if !isUpdated_server_names || !isUpdated_static_myserver || !isUpdated_stamp {
		return fmt.Errorf("failed to update configuration from template file")
	}

	err = os.WriteFile(configFileOut, []byte(out.String()), 0600) // read only for owner
	if err != nil {
		return err
	}

	return nil
}
