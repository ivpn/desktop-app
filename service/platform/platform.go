package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// WrongExecutableFilePermssionsMask - file permissions mask for executables which are not allowed. Executable files should not have write access for someone else except root
	WrongExecutableFilePermissionsMask os.FileMode = 0022
	// DefaultFilePermissionForConfig - mutable config files should have permissions read/write only for owner (root)
	DefaultFilePermissionForConfig os.FileMode = 0600
	// DefaultFilePermissionForStaticConfig - unmutable config files should have permissions read/write only for owner (root)
	DefaultFilePermissionForStaticConfig os.FileMode = 0400
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
func Init() (warnings []string, errors []error) {

	obfsproxyHostPort = 5145

	// do variables initialization for current OS
	warnings, errors = doOsInit()
	if errors == nil {
		errors = make([]error, 0)
	}
	if warnings == nil {
		warnings = make([]string, 0)
	}

	// creating required folders
	if err := makeDir("servicePortFile", filepath.Dir(servicePortFile)); err != nil {
		errors = append(errors, err)
	}
	if err := makeDir("logFile", filepath.Dir(logFile)); err != nil {
		errors = append(errors, err)
	}
	if err := makeDir("openvpnLogFile", filepath.Dir(openvpnLogFile)); err != nil {
		errors = append(errors, err)
	}
	if err := makeDir("settingsFile", filepath.Dir(settingsFile)); err != nil {
		errors = append(errors, err)
	}
	if err := makeDir("openvpnConfigFile", filepath.Dir(openvpnConfigFile)); err != nil {
		errors = append(errors, err)
	}
	if err := makeDir("wgConfigFilePath", filepath.Dir(wgConfigFilePath)); err != nil {
		errors = append(errors, err)
	}

	// checking file permissions
	if _, err := IsFileExists("openvpnCaKeyFile", openvpnCaKeyFile, DefaultFilePermissionForStaticConfig); err != nil {
		errors = append(errors, err)
	}
	if _, err := IsFileExists("openvpnTaKeyFile", openvpnTaKeyFile, DefaultFilePermissionForStaticConfig); err != nil {
		errors = append(errors, err)
	}

	if err := CheckExecutableRights("openvpnUpScript", openvpnUpScript); err != nil {
		errors = append(errors, err)
	}
	if err := CheckExecutableRights("openvpnDownScript", openvpnUpScript); err != nil {
		errors = append(errors, err)
	}

	// checking availability of OpenVPN binaries
	if err := CheckExecutableRights("openVpnBinaryPath", openVpnBinaryPath); err != nil {
		warnings = append(warnings, fmt.Errorf("OpenVPN functionality not accessible: %w", err).Error())
	}
	// checking availability of obfsproxy binaries
	if err := CheckExecutableRights("obfsproxyStartScript", obfsproxyStartScript); err != nil {
		warnings = append(warnings, fmt.Errorf("obfsproxy functionality not accessible: %w", err).Error())
	}
	// checling availability of WireGuard binaries
	if err := CheckExecutableRights("wgBinaryPath", wgBinaryPath); err != nil {
		warnings = append(warnings, fmt.Errorf("WireGuard functionality not accessible: %w", err).Error())
	}
	if err := CheckExecutableRights("wgToolBinaryPath", wgToolBinaryPath); err != nil {
		warnings = append(warnings, fmt.Errorf("WireGuard functionality not accessible: %w", err).Error())
	}
	return warnings, errors
}

// IsFileExists checking file avvailability. If file available - return nil
// When expected file permissin defined (fileMode!=0) - returns error in case if file permission not equal to expected
func IsFileExists(description string, file string, fileMode os.FileMode) (os.FileMode, error) {
	if len(description) > 0 {
		description = fmt.Sprintf("(%s)", description)
	}

	if len(file) == 0 {
		return 0, fmt.Errorf("parameter not initialized %s", description)
	}

	stat, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("file not exists %s: '%s'", description, file)
		}
		return 0, fmt.Errorf("file existing check error %s '%s' : %w", description, file, err)
	}
	mode := stat.Mode()

	if stat.IsDir() {
		return mode, fmt.Errorf("'%s' is directory %s", file, description)
	}

	if fileMode != 0 {

		if mode != fileMode {
			return mode, fmt.Errorf(fmt.Sprintf("file '%s' %s has wrong permissions (%o but expected %o)", file, description, mode, fileMode))
		}
	}

	return mode, nil
}

// CheckExecutableRights checks file has access permission to be writable
// If file not exists or it can be writable by someone alse except root - retun error
func CheckExecutableRights(description string, file string) error {
	if len(file) > 0 {
		// 'file' can be presented as executable with arguments (e.g. 'dns.sh -up')
		// Trying here to take only executable file path without arguments
		file = strings.Split(file, " -")[0]
		file = strings.Split(file, "\t-")[0]
	}

	mode, err := IsFileExists(description, file, 0)
	if err != nil {
		return err
	}

	if len(description) > 0 {
		description = fmt.Sprintf("(%s)", description)
	}

	if (mode & WrongExecutableFilePermissionsMask) > 0 {
		return fmt.Errorf("file '%s' %s has permissins to be modifyied by everyone %o", file, description, mode)
	}
	return nil
}

func makeDir(description string, dirpath string) error {
	if len(dirpath) == 0 {
		return fmt.Errorf("parameter not initialized: %s", description)
	}

	if err := os.MkdirAll(dirpath, os.ModePerm); err != nil {
		return fmt.Errorf("unable to create directory error: %s (%s:%s)", err.Error(), description, dirpath)
	}
	return nil
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
