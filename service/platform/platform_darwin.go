package platform

import (
	"os"
	"path"
)

const (
	// WrongExecutableFilePermissionsMask - file permissions mask for executables which are not allowed. Executable files should not have write access for someone else except root
	WrongExecutableFilePermissionsMask os.FileMode = 0022
	// DefaultFilePermissionForConfig - mutable config files should have permissions read/write only for owner (root)
	DefaultFilePermissionForConfig os.FileMode = 0600
	// DefaultFilePermissionForStaticConfig - unmutable config files should have permissions read/write only for owner (root)
	DefaultFilePermissionForStaticConfig os.FileMode = 0400
)

var (
	firewallScript string
	dnsScript      string
)

// initialize all constant values (e.g. servicePortFile) which can be used in external projects (IVPN CLI)
func doInitConstants() {
	servicePortFile = "/Library/Application Support/IVPN/port.txt"

	logDir := "/Library/Logs/"
	logFile = path.Join(logDir, "IVPN Agent.log")
	openvpnLogFile = path.Join(logDir, "openvpn.log")
}

func doOsInit() (warnings []string, errors []error) {
	warnings, errors = doOsInitForBuild()

	if errors == nil {
		errors = make([]error, 0)
	}

	if err := CheckExecutableRights("firewallScript", firewallScript); err != nil {
		errors = append(errors, err)
	}
	if err := CheckExecutableRights("dnsScript", dnsScript); err != nil {
		errors = append(errors, err)
	}

	return warnings, errors
}

// FirewallScript returns path to firewal script
func FirewallScript() string {
	return firewallScript
}

// DNSScript returns path to DNS script
func DNSScript() string {
	return dnsScript
}
