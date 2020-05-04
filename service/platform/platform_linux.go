package platform

import (
	"path"
)

var (
	firewallScript string
	logDir         string = "/opt/ivpn/log"
	tmpDir         string = "/opt/ivpn/mutable"
)

// initialize all constant values (e.g. servicePortFile) which can be used in external projects (IVPN CLI)
func doInitConstants() {
	servicePortFile = path.Join(tmpDir, "port.txt")

	logFile = path.Join(logDir, "IVPN_Agent.log")
	openvpnLogFile = path.Join(logDir, "openvpn.log")
}

func doOsInit() (warnings []string, errors []error) {
	openVpnBinaryPath = path.Join("/usr/sbin", "openvpn")
	obfsproxyStartScript = "/usr/bin/obfsproxy"
	wgBinaryPath = path.Join("/usr/bin", "wg-quick")
	wgToolBinaryPath = path.Join("/usr/bin", "wg")

	warnings, errors = doOsInitForBuild()

	if errors == nil {
		errors = make([]error, 0)
	}

	if err := checkFileAccessRigthsExecutable("firewallScript", firewallScript); err != nil {
		errors = append(errors, err)
	}

	return warnings, errors
}

func doInitOperations() (w string, e error) { return "", nil }

// FirewallScript returns path to firewal script
func FirewallScript() string {
	return firewallScript
}
