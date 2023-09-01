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

package platform

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ivpn/desktop-app/daemon/helpers"
	"github.com/ivpn/desktop-app/daemon/service/platform/filerights"
)

var (
	// paranoidModeSecretFile path to a file which contains 'secret' (password) for 'Paranoid mode'
	// If 'paranoid mode' enabled - this 'secret' must be used in each request to a daemon.
	// This file should be accessible to read only for 'privilaged' user
	paranoidModeSecretFile string

	settingsFile    string
	servicePortFile string
	serversFile     string
	logFile         string

	openVpnBinaryPath     string
	openvpnCaKeyFile      string
	openvpnTaKeyFile      string
	openvpnConfigFile     string
	openvpnUpScript       string
	openvpnDownScript     string
	openvpnProxyAuthFile  string
	openvpnUserParamsFile string

	obfsproxyStartScript string

	v2rayBinaryPath    string
	v2rayConfigTmpFile string

	routeCommand string // Example: "/sbin/route" - for macOS, "/sbin/ip route" - for Linux, "C:\\Windows\\System32\\ROUTE.EXE" - for Windows

	wgBinaryPath     string
	wgToolBinaryPath string
	wgConfigFilePath string

	kemHelperBinaryPath string

	dnscryptproxyBinPath        string
	dnscryptproxyConfigTemplate string
	dnscryptproxyConfig         string
	dnscryptproxyLog            string
)

func init() {
	// initialize all constant values (e.g. servicePortFile) which can be used in external projects (IVPN CLI)
	doInitConstants()
	if len(servicePortFile) <= 0 {
		panic("Path to service port file not defined ('platform.servicePortFile' is empty)")
	}
}

// Init - initialize all preferences required for a daemon
// Must be called on beginning of application start by a daemon(service)
func Init() (warnings []string, errors []error, logInfo []string) {

	// do variables initialization for current OS
	warnings, errors, logInfo = doOsInit()
	if errors == nil {
		errors = make([]error, 0)
	}
	if warnings == nil {
		warnings = make([]string, 0)
	}

	// creating required folders
	if err := makeDir("servicePortFile", filepath.Dir(servicePortFile), os.ModePerm); err != nil {
		errors = append(errors, err)
	}
	if err := makeDir("paranoidModeSecretFile", filepath.Dir(paranoidModeSecretFile), os.ModePerm); err != nil {
		errors = append(errors, err)
	}
	if err := makeDir("logFile", filepath.Dir(logFile), os.ModePerm); err != nil {
		errors = append(errors, err)
	}
	if err := makeDir("settingsFile", filepath.Dir(settingsFile), os.ModePerm); err != nil {
		errors = append(errors, err)
	}
	if err := makeDir("openvpnConfigFile", filepath.Dir(openvpnConfigFile), os.ModePerm); err != nil {
		errors = append(errors, err)
	}
	if err := makeDir("wgConfigFilePath", filepath.Dir(wgConfigFilePath), os.ModePerm); err != nil {
		errors = append(errors, err)
	}

	// checking file permissions
	if err := checkFileAccessRightsStaticConfig("openvpnCaKeyFile", openvpnCaKeyFile); err != nil {
		errors = append(errors, err)
	}
	if err := checkFileAccessRightsStaticConfig("openvpnTaKeyFile", openvpnTaKeyFile); err != nil {
		errors = append(errors, err)
	}

	if len(openvpnUpScript) > 0 {
		if err := checkFileAccessRightsExecutable("openvpnUpScript", openvpnUpScript); err != nil {
			errors = append(errors, err)
		}
	}

	if len(openvpnDownScript) > 0 {
		if err := checkFileAccessRightsExecutable("openvpnDownScript", openvpnDownScript); err != nil {
			errors = append(errors, err)
		}
	}

	// checking availability of OpenVPN binaries
	if err := checkFileAccessRightsExecutable("openVpnBinaryPath", openVpnBinaryPath); err != nil {
		warnings = append(warnings, fmt.Errorf("OpenVPN functionality not accessible: %w", err).Error())
	}
	// checking availability of obfsproxy binaries
	if err := checkFileAccessRightsExecutable("obfsproxyStartScript", obfsproxyStartScript); err != nil {
		warnings = append(warnings, fmt.Errorf("obfsproxy functionality not accessible: %w", err).Error())
	}
	if err := checkFileAccessRightsExecutable("v2rayBinaryPath", v2rayBinaryPath); err != nil {
		warnings = append(warnings, fmt.Errorf("v2ray functionality not accessible: %w", err).Error())
	}

	// checling availability of WireGuard binaries
	if err := checkFileAccessRightsExecutable("wgBinaryPath", wgBinaryPath); err != nil {
		warnings = append(warnings, fmt.Errorf("WireGuard functionality not accessible: %w", err).Error())
	}
	if err := checkFileAccessRightsExecutable("wgToolBinaryPath", wgToolBinaryPath); err != nil {
		warnings = append(warnings, fmt.Errorf("WireGuard functionality not accessible: %w", err).Error())
	}

	if err := checkFileAccessRightsExecutable("kemHelperBinaryPath", kemHelperBinaryPath); err != nil {
		warnings = append(warnings, fmt.Errorf("KEM functionality not accessible: %w", err).Error())
	}

	if err := checkFileAccessRightsExecutable("dnscryptproxyBinPath", dnscryptproxyBinPath); err != nil {
		errors = append(errors, err)
	}
	if err := checkFileAccessRightsStaticConfig("dnscryptproxyConfigTemplate", dnscryptproxyConfigTemplate); err != nil {
		errors = append(errors, err)
	}

	if len(routeCommand) > 0 {
		routeBinary := strings.Split(routeCommand, " ")[0]
		if err := checkFileAccessRightsExecutable("routeCommand", routeBinary); err != nil {
			routeCommand = ""
			warnings = append(warnings, fmt.Errorf("route binary error: %w", err).Error())
		}
	}

	w, e := doInitOperations()
	if len(w) > 0 {
		warnings = append(warnings, w)
	}
	if e != nil {
		errors = append(errors, e)
	}

	createOpenVpnUserParamsFileExample()

	return warnings, errors, logInfo
}

func checkFileAccessRightsStaticConfig(paramName string, file string) error {
	if err := filerights.CheckFileAccessRightsStaticConfig(file); err != nil {
		return fmt.Errorf("(%s) %w", paramName, err)
	}
	return nil
}

func checkFileAccessRightsExecutable(paramName string, file string) error {
	if err := filerights.CheckFileAccessRightsExecutable(file); err != nil {
		return fmt.Errorf("(%s) %w", paramName, err)
	}
	return nil
}

func createOpenVpnUserParamsFileExample() error {
	if len(openvpnUserParamsFile) <= 0 {
		return nil // openvpnUserParamsFile is not defined
	}

	if helpers.FileExists(openvpnUserParamsFile) {
		if err := filerights.CheckFileAccessRightsConfig(openvpnUserParamsFile); err == nil {
			return nil // file is exists with correct permissions
		}
		// 'openvpnUserParamsFile' has wrong permissions. Removing it.
		os.Remove(openvpnUserParamsFile)
	}

	if err := makeDir("openvpnUserParamsFile", filepath.Dir(openvpnUserParamsFile), os.ModePerm); err != nil {
		return err
	}

	var builder strings.Builder
	builder.WriteString("# This file is created automatically.\n")
	builder.WriteString("# Do not change it's access permissions or ownership!\n")
	builder.WriteString("# You will need administrator permission to edit this file.\n")
	builder.WriteString("# \n")
	builder.WriteString("# This file contains additional user-defined parameters for OpenVPN configuration.\n")
	builder.WriteString("# All parameters defined here will be added to default OpenVPN configuration used by the IVPN Client.\n")
	builder.WriteString("# All changes are made at your own risk!\n")
	builder.WriteString("# We recommend keeping this file empty.\n")

	return ioutil.WriteFile(openvpnUserParamsFile, []byte(builder.String()), filerights.DefaultFilePermissionsForConfig())
}

func makeDir(description string, dirpath string, mode fs.FileMode) error {
	if len(dirpath) == 0 {
		return fmt.Errorf("parameter not initialized: %s", description)
	}

	if err := os.MkdirAll(dirpath, mode); err != nil {
		return fmt.Errorf("unable to create directory error: %s (%s:%s)", err.Error(), description, dirpath)
	}
	return nil
}

// Is64Bit - returns 'true' if binary compiled in 64-bit architecture
func Is64Bit() bool {
	return strconv.IntSize == 64
}

// SettingsFile path to settings file
func SettingsFile() string {
	return settingsFile
}

// ServicePortFile path to service port file
func ServicePortFile() string {
	return servicePortFile
}

// ParanoidModeSecretFile path to a file which contains 'secret' (password) for 'Paranoid mode'
// If 'paranoid mode' enabled - this 'secret' must be used in each request to a daemon.
// This file should be accessible to read only for 'privilaged' user
func ParanoidModeSecretFile() string {
	return paranoidModeSecretFile
}

// ServersFile path to servers.json
func ServersFile() string {
	return serversFile
}

// LogFile path to log-file
func LogFile() string {
	return logFile
}

func LogDir() string {
	return filepath.Dir(logFile)
}

// OpenVpnBinaryPath path to openvpn binary
func OpenVpnBinaryPath() string {
	return openVpnBinaryPath
}

// OpenvpnCaKeyFile path to openvpn CA key file
func OpenvpnCaKeyFile() string {
	return openvpnCaKeyFile
}

// OpenvpnTaKeyFile path to openvpn TA key file
func OpenvpnTaKeyFile() string {
	return openvpnTaKeyFile
}

// OpenvpnConfigFile path to openvpn config file
func OpenvpnConfigFile() string {
	return openvpnConfigFile
}

// OpenvpnUpScript path to openvpn UP script file
func OpenvpnUpScript() string {
	return openvpnUpScript
}

// OpenvpnDownScript path to openvpn Down script file
func OpenvpnDownScript() string {
	return openvpnDownScript
}

// OpenvpnProxyAuthFile path to openvpn proxy credentials file
func OpenvpnProxyAuthFile() string {
	return openvpnProxyAuthFile
}

// OpenvpnUserParamsFile returns a path to a user-defined extra peremeters for OpenVPN configuration
func OpenvpnUserParamsFile() string {
	return openvpnUserParamsFile
}

// ObfsproxyStartScript path to obfsproxy binary
func ObfsproxyStartScript() string {
	return obfsproxyStartScript
}

func V2RayBinaryPath() string {
	return v2rayBinaryPath
}
func V2RayConfigFile() string {
	return v2rayConfigTmpFile
}

// RouteCommand shell command to update routing table
// Example: "/sbin/route" - for macOS, "/sbin/ip route" - for Linux, "C:\\Windows\\System32\\ROUTE.EXE" - for Windows
func RouteCommand() string {
	return routeCommand
}

// WgBinaryPath path to WireGuard binary
func WgBinaryPath() string {
	return wgBinaryPath
}

// WgToolBinaryPath path to WireGuard tools binary
func WgToolBinaryPath() string {
	return wgToolBinaryPath
}

// WGConfigFilePath path to WireGuard configuration file
func WGConfigFilePath() string {
	return wgConfigFilePath
}

func DnsCryptProxyInfo() (binPath, configPathTemplate, configPathMutable, logPath string) {
	return dnscryptproxyBinPath, dnscryptproxyConfigTemplate, dnscryptproxyConfig, dnscryptproxyLog
}

func KemHelperBinaryPath() string {
	return kemHelperBinaryPath
}
