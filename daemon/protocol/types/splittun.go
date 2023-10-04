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

package types

import (
	"github.com/ivpn/desktop-app/daemon/oshelpers"
	"github.com/ivpn/desktop-app/daemon/splittun"
)

// GetInstalledApps (request) requests information about installed applications on the system
type GetInstalledApps struct {
	RequestBase
	// (optional) Platform-depended: extra parameters (in JSON)
	// For Windows:
	//			WindowsEnvAppdata 	string
	// 				Applicable only for Windows: APPDATA environment variable
	// 				Needed to know path of current user's (not root) StartMenu folder location
	// For Linux:
	//			EnvVar_XDG_CURRENT_DESKTOP string
	//			EnvVar_XDG_DATA_DIRS       string
	//			EnvVar_HOME                string
	//			IconsTheme                 string
	ExtraArgsJSON string
}

// InstalledAppsResp (response) contains information about installed applications on the system
type InstalledAppsResp struct {
	CommandBase
	Apps []oshelpers.AppInfo
}

// GetAppIcon (request) requests shell icon for binary file (application)
// Note: ensure if SplitTunnelStatus.IsCanGetAppIconForBinary is active
type GetAppIcon struct {
	RequestBase
	AppBinaryPath string
}

// AppIconResp (response) contains information about shell icon for binary file (application)
type AppIconResp struct {
	CommandBase
	AppBinaryPath string
	AppIcon       string // base64 png image
}

// SplitTunnelSet (request) sets the split-tunnelling configuration
type SplitTunnelSetConfig struct {
	RequestBase
	IsEnabled        bool // is ST enabled
	IsInversed       bool // when inversed - only apps added to ST will use VPN connection, all other apps will use direct unencrypted connection
	IsAnyDns         bool // (only for Inverse Split Tunnel) When false: Allow only DNS servers specified by the IVPN application
	IsAllowWhenNoVpn bool // (only for Inverse Split Tunnel) Allow connectivity for Split Tunnel apps when VPN is disabled
	Reset            bool // disable ST and erase all ST config (if enabled - all the rest paremeters are ignored)
}

// GetSplitTunnelStatus (request) requests the Split-Tunnelling configuration
type SplitTunnelGetStatus struct {
	RequestBase
}

// SplitTunnelStatus (response) returns the split-tunnelling configuration
type SplitTunnelStatus struct {
	CommandBase

	IsEnabled        bool // is ST enabled
	IsInversed       bool // Inverse Split Tunnel (only 'splitted' apps use VPN tunnel)
	IsAnyDns         bool // (only for Inverse Split Tunnel) When false: Allow only DNS servers specified by the IVPN application
	IsAllowWhenNoVpn bool // (only for Inverse Split Tunnel) Allow connectivity for Split Tunnel apps when VPN is disabled

	IsFunctionalityNotAvailable bool // TODO: this is redundant, remove it. Use daemon disabled functions info instead (Note: it is in use by CLI project)
	// This parameter informs availability of the functionality to get icon for particular binary
	// (true - if commands GetAppIcon/AppIconResp  applicable for this platform)
	IsCanGetAppIconForBinary bool
	// Information about applications added to ST configuration
	// (applicable for Windows)
	SplitTunnelApps []string
	// Information about active applications running in Split-Tunnel environment
	// (applicable for Linux)
	RunningApps []splittun.RunningApp
}

// SplitTunnelAddApp (request) add application to SplitTunneling
// Expected response:
//
//			Windows	- types.EmptyResp (success)
//	 	Linux	- SplitTunnelAddAppCmdResp -> contains shell command which have to be executed in user space environment
//
// Description of Split Tunneling commands sequence to run the application:
//
//		[client]					[daemon]
//		SplitTunnelAddApp		->
//								<-	windows:	types.EmptyResp (success)
//								<-	linux:		types.SplitTunnelAddAppCmdResp (some operations required on client side)
//		<windows: done>
//		<execute shell command: types.SplitTunnelAddAppCmdResp.CmdToExecute and get PID>
//	 SplitTunnelAddedPidInfo	->
//								<-	types.EmptyResp (success)
type SplitTunnelAddApp struct {
	RequestBase
	// Windows: full path to the app binary
	// Linux: command to be executed in ST environment (e.g. binary + arguments)
	Exec string
}

// SplitTunnelAddAppCmdResp (response) contains shell command which have to be executed in user space environment
// (not in use for Windows platform)
type SplitTunnelAddAppCmdResp struct {
	CommandBase
	// Command will be executed in ST environment
	// (identical to SplitTunnelAddApp.Exec)
	Exec string
	// Shell command which have to be executed in user space environment
	CmdToExecute string

	IsAlreadyRunning        bool
	IsAlreadyRunningMessage string
}

// SplitTunnelAddedPidInfo (request) informs the daemon about started process in ST environment
// (not in use for Windows platform)
type SplitTunnelAddedPidInfo struct {
	RequestBase
	Pid int
	// Command will be executed in ST environment (e.g. binary + arguments)
	// (identical to SplitTunnelAddApp.Exec and SplitTunnelAddAppCmdResp.Exec)
	Exec string
	// Shell command used to perform this operation
	CmdToExecute string
}

type SplitTunnelRemoveApp struct {
	RequestBase
	// (applicable for Linux) PID of the running process in ST environment
	Pid int
	// (applicable for Windows) full path to the app binary to be excluded from ST
	Exec string
}
