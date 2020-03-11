package platform

var (
	firewallScript string
	dnsScript      string
)

// initialize all constant values (e.g. servicePortFile) which can be used in external projects (IVPN CLI)
func doInitConstants() {
	servicePortFile = "/Library/Application Support/IVPN/port.txt"
}

func doOsInit() {
	doOsInitForBuild()
	ensureFileExists("firewallScript", firewallScript)
	ensureFileExists("dnsScript", dnsScript)
}

// FirewallScript returns path to firewal script
func FirewallScript() string {
	return firewallScript
}

// DNSScript returns path to DNS script
func DNSScript() string {
	return dnsScript
}
