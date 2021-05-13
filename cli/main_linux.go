//
//  IVPN command line interface (CLI)
//  https://github.com/ivpn/desktop-app-cli
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

package main

import (
	"fmt"
	"io/ioutil"
	"path"
)

func printServStartInstructions() {
	fmt.Printf("Please, restart 'ivpn-service'\n")
	tmpDir := "/opt/ivpn/mutable"
	// print service install instructions (if exists)
	content, err := ioutil.ReadFile(path.Join(tmpDir, "service_install.txt"))
	if err == nil {
		fmt.Println(string(content))
	}
}
