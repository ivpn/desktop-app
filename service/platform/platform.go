package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

var (
	settingsFile    string
	servicePortFile string
	serversFile     string
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
	// initialize all constant values (e.g. servicePortFile) which can be used in external projects (IVPN CLI)
	doInitConstants()
	if len(servicePortFile) <= 0 {
		panic("Path to service port file not defined ('platform.servicePortFile' is empty)")
	}
}

// Init - initialize all preferences required for a daemon
// Must be called on beginning of application start by a daemon(service)
func Init() {
	obfsproxyHostPort = 5145

	// do variables initialization for current OS
	doOsInit()
	makeDir("logFile", filepath.Dir(logFile))
	makeDir("settingsFile", filepath.Dir(settingsFile))

	// checling availability of OpenVPN binaries
	panicIfFileNotExists("openvpnCaKeyFile", openvpnCaKeyFile)
	panicIfFileNotExists("openvpnTaKeyFile", openvpnTaKeyFile)
	if err := isFileExists("openVpnBinaryPath", openVpnBinaryPath); err != nil {
		fmt.Println(fmt.Errorf("WARNING! OpenVPN functionality not accessible: %w", err))
	}

	// checling availability of obfsproxy binaries
	if err := isFileExists("obfsproxyStartScript", obfsproxyStartScript); err != nil {
		fmt.Println(fmt.Errorf("WARNING! obfsproxy functionality not accessible: %w", err))
	}

	// checling availability of WireGuard binaries
	if err := isFileExists("wgBinaryPath", wgBinaryPath); err != nil {
		fmt.Println(fmt.Errorf("WARNING! WireGuard functionality not accessible: %w", err))
	}
	if err := isFileExists("wgToolBinaryPath", wgToolBinaryPath); err != nil {
		fmt.Println(fmt.Errorf("WARNING! WireGuard functionality not accessible: %w", err))
	}
}

func panicIfFileNotExists(description string, file string) {
	if err := isFileExists(description, file); err != nil {
		panic(fmt.Errorf("[Initialisation (plafrorm)] %w", err))
	}
}

func isFileExists(description string, file string) (err error) {
	defer func() {
		if err != nil {
			fmt.Printf("[Initialisation (plafrorm)] %s\n", err)
		}
	}()

	if len(file) == 0 {
		return fmt.Errorf("parameter not initialized: %s", description)
	}

	if _, err := os.Stat(file); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not exists (%s): '%s'", description, file)
		}
		return fmt.Errorf("file existing check error: %s (%s:%s)", err.Error(), description, file)
	}
	return nil
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
