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

package oshelpers

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ivpn/desktop-app/daemon/logger"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("oshlpr")
}

type AppInfo struct {
	// Application description: [<AppGroup>/]<AppName>.
	// Example 1: "Git/Git GUI"
	// 		AppName  = "Git GUI"
	// 		AppGroup = "Git"
	// Example 2: "Firefox"
	// 		AppName  = "Firefox"
	// 		AppGroup = null
	AppName  string
	AppGroup string // optional
	// base64 icon of the executable binary
	AppIcon string
	// The unique parameter describing an application
	// Windows: absolute path to application binary
	// Linux: program to execute, possibly with arguments.
	AppBinaryPath string
}

// GetInstalledApps returns a list of installed applications on the system
// Important! All elements in the return list should have unique AppBinaryPath!
// Parameters:
//
//	extraArgsJSON - (optional) Platform-depended: extra parameters (in JSON)
//	For Windows:
//		{ "WindowsEnvAppdata": "..." }
//		Applicable only for Windows: APPDATA environment variable
//		Needed to know path of current user's (not root) StartMenu folder location
func GetInstalledApps(extraArgsJSON string) (apps []AppInfo, err error) {
	defer func() {
		if r := recover(); r != nil {
			apps = nil
			if theErr, ok := r.(error); ok {
				err = fmt.Errorf("PANIC on GetInstalledApps() [recovered] : %w", theErr)
			} else {
				err = fmt.Errorf("PANIC on GetInstalledApps() [recovered] ")
			}
			log.Error(err)
		}
	}()

	appsList, err := implGetInstalledApps(extraArgsJSON)
	if err != nil {
		return appsList, err
	}

	// ensure AppBinaryPath is unique for all elements in the list
	retMap := make(map[string]struct{})
	retAppsList := make([]AppInfo, 0, len(appsList))
	for _, v := range appsList {
		if _, ok := retMap[v.AppBinaryPath]; ok {
			continue // duplicate
		}
		retMap[v.AppBinaryPath] = struct{}{}
		retAppsList = append(retAppsList, v)
	}

	// sort by app name
	sort.Slice(retAppsList[:], func(i, j int) bool {
		return strings.Compare(retAppsList[i].AppName, retAppsList[j].AppName) == -1
	})

	return retAppsList, nil
}

func GetBinaryIconBase64(binaryPath string) (icon string, err error) {
	defer func() {
		if r := recover(); r != nil {
			icon = ""
			if theErr, ok := r.(error); ok {
				err = fmt.Errorf("PANIC on GetBinaryBase64PngIcon() [recovered] : %w", theErr)
			} else {
				err = fmt.Errorf("PANIC on GetBinaryIconBase64() [recovered] ")
			}
			log.Error(err)
		}
	}()

	f := implGetFunc_BinaryIconBase64()
	if f == nil {
		return "", fmt.Errorf("not implemented for this platform")
	}
	return f(binaryPath)
}

// IsCanGetAppIconForBinary informs availability of the functionality to get icon for particular binary
// (true - if function GetBinaryIconBase64() applicable for this platform)
func IsCanGetAppIconForBinary() bool {
	return implGetFunc_BinaryIconBase64() != nil
}
