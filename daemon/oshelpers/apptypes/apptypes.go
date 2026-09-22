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

// Package apptypes holds the data types shared between the daemon's
// installed-applications helpers (oshelpers) and the client protocol
// (protocol/types). It imports nothing, so the protocol package - and with it
// every client of it, such as the CLI - never depends on the platform
// implementation behind oshelpers.
package apptypes

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
	// macOS: absolute path to the application bundle or executable
	AppBinaryPath string
}
