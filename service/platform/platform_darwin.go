package platform

var (
	firewallScript string
	dnsScript      string
)

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
