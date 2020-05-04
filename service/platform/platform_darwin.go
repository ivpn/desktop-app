package platform

import (
	"path"
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

	if err := checkFileAccessRigthsExecutable("firewallScript", firewallScript); err != nil {
		errors = append(errors, err)
	}
	if err := checkFileAccessRigthsExecutable("dnsScript", dnsScript); err != nil {
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
