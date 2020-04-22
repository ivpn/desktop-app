package types

// WireGuardServerHostInfo contains info about WG server host
type WireGuardServerHostInfo struct {
	Hostname  string `json:"hostname"`
	Host      string `json:"host"`
	PublicKey string `json:"public_key"`
	LocalIP   string `json:"local_ip"`
}

// WireGuardServerInfo contains all info about WG server
type WireGuardServerInfo struct {
	Gateway     string `json:"gateway"`
	CountryCode string `json:"country_code"`
	Country     string `json:"country"`
	City        string `json:"city"`

	Hosts []WireGuardServerHostInfo `json:"hosts"`
}

// OpenvpnServerInfo contains all info about OpenVPN server
type OpenvpnServerInfo struct {
	Gateway     string   `json:"gateway"`
	CountryCode string   `json:"country_code"`
	Country     string   `json:"country"`
	City        string   `json:"city"`
	IPAddresses []string `json:"ip_addresses"`
}

// DNSInfo contains info about DNS server
type DNSInfo struct {
	IP         string `json:"ip"`
	MultihopIP string `json:"multihop-ip"`
}

// AntitrackerInfo all info about antitracker DNSs
type AntitrackerInfo struct {
	Default  DNSInfo `json:"default"`
	Hardcore DNSInfo `json:"hardcore"`
}

// InfoAPI contains API IP adresses
type InfoAPI struct {
	IPAddresses []string `json:"ips"`
}

// ConfigInfo contains different configuration info (Antitracker, API ...)
type ConfigInfo struct {
	Antitracker AntitrackerInfo `json:"antitracker"`
	API         InfoAPI         `json:"api"`
}

// ServersInfoResponse all info from servers.json
type ServersInfoResponse struct {
	WireguardServers []WireGuardServerInfo `json:"wireguard"`
	OpenvpnServers   []OpenvpnServerInfo   `json:"openvpn"`
	Config           ConfigInfo            `json:"config"`
}
