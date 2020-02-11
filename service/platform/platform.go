package platform

import (
	"fmt"
	"os"
	"strconv"
)

var (
	settingsDir     string
	settingsFile    string
	servicePortFile string
	serversFile     string
	logDir          string
	logFile         string
	openvpnLogFile  string

	openVpnBinaryPath    string
	openvpnCaKeyFile     string
	openvpnTaKeyFile     string
	openvpnConfigFile    string
	openvpnUpScript      string
	openvpnDownScript    string
	openvpnProxyAuthFile string

	obfsproxyStartScript string
	obfsproxyHostPort    int

	wgBinaryPath     string
	wgToolBinaryPath string
	wgConfigFilePath string
)

func init() {
	obfsproxyHostPort = 5145

	// do variables initialization for current OS
	doOsInit()

	ensureFileExists("openVpnBinaryPath", openVpnBinaryPath)
	ensureFileExists("openvpnCaKeyFile", openvpnCaKeyFile)
	ensureFileExists("openvpnTaKeyFile", openvpnTaKeyFile)

	ensureFileExists("obfsproxyStartScript", obfsproxyStartScript)

	ensureFileExists("wgBinaryPath", wgBinaryPath)
	ensureFileExists("wgToolBinaryPath", wgToolBinaryPath)

	makeDir("settingsDir", settingsDir)
	makeDir("logDir", logDir)
}

func ensureFileExists(description string, file string) {
	if len(file) == 0 {
		panic(fmt.Sprintf("[Initialisation (plafrorm)] Parameter not initialized: %s", description))
	}

	if _, err := os.Stat(file); err != nil {
		if os.IsNotExist(err) {
			panic(fmt.Sprintf("[Initialisation (plafrorm)] File not exists (%s): '%s'", description, file))
		} else {
			panic(fmt.Sprintf("[Initialisation (plafrorm)] File existing check error: %s (%s:%s)", err.Error(), description, file))
		}
	}
}

func makeDir(description string, dirpath string) {
	if len(dirpath) == 0 {
		panic(fmt.Sprintf("[Initialisation (plafrorm)] Parameter not initialized: %s", description))
	}

	if err := os.MkdirAll(dirpath, os.ModePerm); err != nil {
		panic(fmt.Sprintf("[Initialisation (plafrorm)] Unable to create directory error: %s (%s:%s)", err.Error(), description, dirpath))
	}
}

// Is64Bit - returns 'true' if binary compiled in 64-bit architecture
func Is64Bit() bool {
	if strconv.IntSize == 64 {
		return true
	}
	return false
}

// SettingsFile path to settings file
func SettingsFile() string {
	return settingsFile
}

// ServicePortFile parh to service port file
func ServicePortFile() string {
	return servicePortFile
}

// ServersFile path to servers.json
func ServersFile() string {
	return serversFile
}

// LogFile path to log-file
func LogFile() string {
	return logFile
}

// OpenvpnLogFile path to log-file for openvpn
func OpenvpnLogFile() string {
	return openvpnLogFile
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

// ObfsproxyStartScript path to obfsproxy binary
func ObfsproxyStartScript() string {
	return obfsproxyStartScript
}

// ObfsproxyHostPort is an port of obfsproxy host
func ObfsproxyHostPort() int {
	return obfsproxyHostPort
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
