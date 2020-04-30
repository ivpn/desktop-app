package platform

import (
	"fmt"
	"os"
	"path"
)

// !file permissins check is DISABLED FOR WINDOWS!
const (
	// WrongExecutableFilePermissionsMask - file permissions mask for executables which are not allowed. Executable files should not have write access for someone else except root
	WrongExecutableFilePermissionsMask os.FileMode = 0
	// DefaultFilePermissionForConfig - mutable config files should permissions
	DefaultFilePermissionForConfig os.FileMode = 0
	// DefaultFilePermissionForStaticConfig - unmutable config files permissions
	DefaultFilePermissionForStaticConfig os.FileMode = 0
)

var (
	wfpDllPath           string
	nativeHelpersDllPath string
)

func doInitConstants() {
	doInitConstantsForBuild()

	installDir := getInstallDir()
	if len(servicePortFile) <= 0 {
		servicePortFile = path.Join(installDir, "etc/port.txt")
	} else {
		// debug version can have different port file value
		fmt.Println("!!! WARNING!!! Non-standard service port file: ", servicePortFile)
	}

	logFile = path.Join(installDir, "log/IVPN Agent.log")
	openvpnLogFile = path.Join(installDir, "log/openvpn.log")
}

func doOsInit() (warnings []string, errors []error) {
	doOsInitForBuild()
	_installDir := getInstallDir()

	_archDir := "x86_64"
	if Is64Bit() == false {
		_archDir = "x86"
	}

	if errors == nil {
		errors = make([]error, 0)
	}

	// common variables initialization
	settingsDir := path.Join(_installDir, "etc")
	settingsFile = path.Join(settingsDir, "settings.json")

	serversFile = path.Join(settingsDir, "servers.json")
	openvpnConfigFile = path.Join(settingsDir, "openvpn.cfg")
	openvpnProxyAuthFile = path.Join(settingsDir, "proxyauth.txt")
	wgConfigFilePath = path.Join(settingsDir, "IVPN.conf") // will be used also for WireGuard service name (e.g. "WireGuardTunnel$IVPN")

	openVpnBinaryPath = path.Join(_installDir, "OpenVPN", _archDir, "openvpn.exe")
	openvpnCaKeyFile = path.Join(settingsDir, "ca.crt")
	openvpnTaKeyFile = path.Join(settingsDir, "ta.key")
	openvpnUpScript = ""
	openvpnDownScript = ""

	obfsproxyStartScript = path.Join(_installDir, "OpenVPN", "obfsproxy", "obfsproxy.exe")

	_wgArchDir := "x86_64"
	if _, err := os.Stat(path.Join(_installDir, "WireGuard", _wgArchDir, "wireguard.exe")); err != nil {
		_wgArchDir = "x86"
		if _, err := os.Stat(path.Join(_installDir, "WireGuard", _wgArchDir, "wireguard.exe")); err != nil {
			errors = append(errors, fmt.Errorf("Unabale to find WireGuard binary: %s ..<x86_64\\x86>", path.Join(_installDir, "WireGuard")))
		}
	}
	wgBinaryPath = path.Join(_installDir, "WireGuard", _wgArchDir, "wireguard.exe")
	wgToolBinaryPath = path.Join(_installDir, "WireGuard", _wgArchDir, "wg.exe")

	if _, err := os.Stat(wfpDllPath); err != nil {
		errors = append(errors, fmt.Errorf("file not exists: '%s'", wfpDllPath))
	}
	if _, err := os.Stat(nativeHelpersDllPath); err != nil {
		errors = append(errors, fmt.Errorf("file not exists: '%s'", nativeHelpersDllPath))
	}

	return warnings, errors
}

func doInitOperations() (w string, e error) { return "", nil }

// WindowsWFPDllPath - Path to Windows DLL with helper methods for WFP (Windows Filtering Platform)
func WindowsWFPDllPath() string {
	return wfpDllPath
}

// WindowsNativeHelpersDllPath - Path to Windows DLL with helper methods (native DNS implementation... etc.)
func WindowsNativeHelpersDllPath() string {
	return nativeHelpersDllPath
}
