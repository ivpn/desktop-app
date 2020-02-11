package platform

import (
	"fmt"
	"os"
	"path"
	"strings"
)

var (
	wfpDllPath           string
	nativeHelpersDllPath string
)

func doOsInit() {
	installDir := doOsInitForBuild()
	initVars(installDir)
}

func initVars(_installDir string) {
	_archDir := "x86_64"
	if Is64Bit() == false {
		_archDir = "x86"
	}

	_installDir = strings.ReplaceAll(_installDir, `\`, `/`)

	// common variables initialization
	settingsDir = path.Join(_installDir, "etc")
	settingsFile = path.Join(settingsDir, "settings.json")
	servicePortFile = path.Join(settingsDir, "port.txt")
	serversFile = path.Join(settingsDir, "servers.json")
	openvpnConfigFile = path.Join(settingsDir, "openvpn.cfg")
	openvpnProxyAuthFile = path.Join(settingsDir, "proxyauth.txt")
	wgConfigFilePath = path.Join(settingsDir, "IVPN.conf") // will be used also for WireGuard service name (e.g. "WireGuardTunnel$IVPN")

	logDir = path.Join(_installDir, "log")
	logFile = path.Join(logDir, "IVPN Agent.log")
	openvpnLogFile = path.Join(logDir, "openvpn.log")

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
			panic(fmt.Sprintf("[Initialisation (plafrorm)] Unabale to find WireGuard binary: %s ..<x86_64\\x86>", path.Join(_installDir, "WireGuard")))
		}
	}
	wgBinaryPath = path.Join(_installDir, "WireGuard", _wgArchDir, "wireguard.exe")
	wgToolBinaryPath = path.Join(_installDir, "WireGuard", _wgArchDir, "wg.exe")

	ensureFileExists("wfpDllPath", wfpDllPath)
	ensureFileExists("nativeHelpersDllPath", nativeHelpersDllPath)
}

// WindowsWFPDllPath - Path to Windows DLL with helper methods for WFP (Windows Filtering Platform)
func WindowsWFPDllPath() string {
	return wfpDllPath
}

// WindowsNativeHelpersDllPath - Path to Windows DLL with helper methods (native DNS implementation... etc.)
func WindowsNativeHelpersDllPath() string {
	return nativeHelpersDllPath
}
