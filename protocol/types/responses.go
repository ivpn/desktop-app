package types

import (
	"encoding/json"

	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/service/api"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("prtres")
}

func marshalRespnse(respObject interface{}) (ret []byte, err error) {
	data, err := json.Marshal(respObject)
	if err != nil {
		log.Error("Error serializing response:", err)
		return nil, err
	}
	return data, nil
}

// IVPNErrorResponse response of error
func IVPNErrorResponse(err error) []byte {
	type IVPNErrorResponse struct {
		RespType     string `json:"$type"`
		ErrorMessage string
	}

	data, _ := marshalRespnse(IVPNErrorResponse{RespType: "IVPN.IVPNErrorResponse, IVPN.Core", ErrorMessage: err.Error()})
	return data
}

// IVPNEmptyResponse empty response on request
func IVPNEmptyResponse() []byte {
	type IVPNEmptyResponse struct {
		RespType string `json:"$type"`
	}

	data, _ := marshalRespnse(IVPNEmptyResponse{RespType: "IVPN.IVPNEmptyResponse, IVPN.Core"})
	return data
}

// IVPNServiceExitingResponse service is going to exit response
func IVPNServiceExitingResponse() []byte {
	type IVPNServiceExitingResponse struct {
		RespType string `json:"$type"`
	}

	data, _ := marshalRespnse(IVPNServiceExitingResponse{RespType: "IVPN.IVPNServiceExitingResponse, IVPN.Core"})
	return data
}

// IVPNHelloResponse response on initial request
func IVPNHelloResponse() []byte {
	type IVPNHelloResponse struct {
		RespType string `json:"$type"`
		Version  string
	}

	data, _ := marshalRespnse(IVPNHelloResponse{RespType: "IVPN.IVPNHelloResponse, IVPN.Core", Version: "1.0"})
	return data
}

// IVPNKillSwitchGetStatusResponse returns kill-switch status
func IVPNKillSwitchGetStatusResponse(status bool) []byte {
	type IVPNKillSwitchGetStatusResponse struct {
		RespType  string `json:"$type"`
		IsEnabled bool
	}

	data, _ := marshalRespnse(IVPNKillSwitchGetStatusResponse{RespType: "IVPN.IVPNKillSwitchGetStatusResponse, IVPN.Core", IsEnabled: status})
	return data
}

// IVPNKillSwitchGetIsPestistentResponse returns kill-switch persistance status
func IVPNKillSwitchGetIsPestistentResponse(status bool) []byte {
	type IVPNKillSwitchGetIsPestistentResponse struct {
		RespType     string `json:"$type"`
		IsPersistent bool
	}

	data, _ := marshalRespnse(IVPNKillSwitchGetIsPestistentResponse{RespType: "IVPN.IVPNKillSwitchGetIsPestistentResponse, IVPN.Core", IsPersistent: status})
	return data
}

// IVPNDiagnosticsGeneratedResponse returns info from daemon logs
func IVPNDiagnosticsGeneratedResponse(servLog string, servLog0 string) []byte {
	type IVPNDiagnosticsGeneratedResponse struct {
		RespType       string `json:"$type"`
		ServiceLog     string
		ServiceLog0    string
		OpenvpnLog     string
		OpenvpnLog0    string
		EnvironmentLog string
	}

	data, _ := marshalRespnse(IVPNDiagnosticsGeneratedResponse{RespType: "IVPN.IVPNDiagnosticsGeneratedResponse, IVPN.Core", ServiceLog: servLog, ServiceLog0: servLog0})
	return data
}

// IVPNSetAlternateDNSResponse returns status of changing DNS
func IVPNSetAlternateDNSResponse(isSuccess bool, newDNS string) []byte {
	type IVPNSetAlternateDNSResponse struct {
		RespType   string `json:"$type"`
		IsSuccess  bool
		ChangedDNS string `json:"ChangedDns"`
	}

	data, _ := marshalRespnse(IVPNSetAlternateDNSResponse{RespType: "IVPN.IVPNSetAlternateDnsResponse, IVPN.Core", IsSuccess: isSuccess, ChangedDNS: newDNS})
	return data
}

// IVPNConnectedResponse notifying about established connection
func IVPNConnectedResponse(timeSecFrom1970 int64, clientIP string, serverIP string) []byte {
	type IVPNConnectedResponse struct {
		RespType        string `json:"$type"`
		TimeSecFrom1970 int64
		ClientIP        string
		ServerIP        string
	}

	data, _ := marshalRespnse(IVPNConnectedResponse{RespType: "IVPN.IVPNConnectedResponse, IVPN.Core",
		TimeSecFrom1970: timeSecFrom1970,
		ClientIP:        clientIP,
		ServerIP:        serverIP})
	return data
}

// IVPNDisconnectedResponse notifying about stopped connetion
func IVPNDisconnectedResponse(failure bool, authrnticationError bool, reasonDescription string) []byte {
	type IVPNDisconnectedResponse struct {
		RespType          string `json:"$type"`
		Failure           bool
		Reason            int
		ReasonDescription string
	}

	reason := 0
	if authrnticationError == true {
		reason = 1
	}

	data, _ := marshalRespnse(IVPNDisconnectedResponse{RespType: "IVPN.IVPNDisconnectedResponse, IVPN.Core",
		Failure:           failure,
		Reason:            reason,
		ReasonDescription: reasonDescription})
	return data
}

// IVPNStateResponse returns VPN connection state
func IVPNStateResponse(state string, additionalInfo string) []byte {
	type IVPNStateResponse struct {
		RespType            string `json:"$type"`
		State               string
		StateAdditionalInfo string
	}

	data, _ := marshalRespnse(IVPNStateResponse{RespType: "IVPN.IVPNStateResponse, IVPN.Core",
		State:               state,
		StateAdditionalInfo: additionalInfo})
	return data
}

// ======================================================
// .NET client servers type conversion
//
// It is necessary to fit ServersInfoResponse to a type which is expected to receive by .NET implementation of IVPN client
// (to keep compatibility with previous implementation of IVPN agent (.NET))
// TODO: Temporary solution. Must be changed in future (on IVPN client side also). Client must expect to receive ServersInfoResponse type
// ======================================================

// IVPNServerListResponse returns list of servers
func IVPNServerListResponse(servers *api.ServersInfoResponse) []byte {

	type NetWireGuardServerHostInfo struct {
		Type string `json:"$type"`

		Hostname  string
		Host      string `json:"Host"`
		PublicKey string `json:"PublicKey"`
		LocalIP   string `json:"LocalIp"`
	}

	type IPAddressesArray struct {
		Type   string   `json:"$type"`
		Values []string `json:"$values"`
	}

	type HostsArray struct {
		Type   string                       `json:"$type"`
		Values []NetWireGuardServerHostInfo `json:"$values"`
	}

	type NetWireGuardServerInfo struct {
		Type string `json:"$type"`

		Gateway     string `json:"GatewayId"`
		CountryCode string `json:"CountryCode"`
		Country     string `json:"Country"`
		City        string `json:"City"`

		Hosts HostsArray `json:"Hosts"`
	}

	type NetOpenvpnServerInfo struct {
		Type string `json:"$type"`

		Gateway     string `json:"GatewayId"`
		CountryCode string `json:"CountryCode"`
		Country     string `json:"Country"`
		City        string `json:"City"`

		IPAddresses IPAddressesArray `json:"IpAddresses"`
	}

	type NetWireGuardServerInfoArray struct {
		Type   string                   `json:"$type"`
		Values []NetWireGuardServerInfo `json:"$values"`
	}

	type NetOpenvpnServerInfoArray struct {
		Type   string                 `json:"$type"`
		Values []NetOpenvpnServerInfo `json:"$values"`
	}

	type NetDNSInfo struct {
		Type string `json:"$type"`

		IP         string `json:"Ip"`
		MultihopIP string `json:"multihop-ip"`
	}

	type NetAntitrackerInfo struct {
		Type string `json:"$type"`

		Default  NetDNSInfo `json:"default"`
		Hardcore NetDNSInfo `json:"hardcore"`
	}

	type NetConfigAPIInfoIPsArray struct {
		Type   string   `json:"$type"`
		Values []string `json:"$values"`
	}

	type NetConfigAPIInfo struct {
		Type string                   `json:"$type"`
		IPs  NetConfigAPIInfoIPsArray `json:"ips"`
	}

	type NetConfigInfo struct {
		Type string `json:"$type"`

		Antitracker NetAntitrackerInfo `json:"antitracker"`
		API         NetConfigAPIInfo   `json:"api"`
	}

	type NetServersInfoResponse struct {
		Type string `json:"$type"`

		WireguardServers NetWireGuardServerInfoArray `json:"WireGuardServers"`
		OpenvpnServers   NetOpenvpnServerInfoArray   `json:"OpenVPNServers"`
		Config           NetConfigInfo               `json:"Config"`
	}

	type IVPNServerListResponse struct {
		Type       string `json:"$type"`
		VpnServers NetServersInfoResponse
	}

	srvrs := NetServersInfoResponse{
		Type: "IVPN.VpnProtocols.VpnServersInfo, IVPN.Core"}

	srvrs.Config = NetConfigInfo{
		Type: "IVPN.RESTApi.RestRequestGetServers+ServersInfoResponse+ConfigInfoResponse, IVPN.Core",
		Antitracker: NetAntitrackerInfo{
			Type:     "IVPN.RESTApi.RestRequestGetServers+ServersInfoResponse+ConfigInfoResponse+AntiTrackerInfo, IVPN.Core",
			Default:  NetDNSInfo{Type: "IVPN.RESTApi.RestRequestGetServers+ServersInfoResponse+ConfigInfoResponse+AntiTrackerInfo+DnsInfo, IVPN.Core", IP: servers.Config.Antitracker.Default.IP, MultihopIP: servers.Config.Antitracker.Default.MultihopIP},
			Hardcore: NetDNSInfo{Type: "IVPN.RESTApi.RestRequestGetServers+ServersInfoResponse+ConfigInfoResponse+AntiTrackerInfo+DnsInfo, IVPN.Core", IP: servers.Config.Antitracker.Hardcore.IP, MultihopIP: servers.Config.Antitracker.Hardcore.MultihopIP}},
		API: NetConfigAPIInfo{
			Type: "IVPN.RESTApi.RestRequestGetServers+ServersInfoResponse+ConfigInfoResponse+ApiInfo, IVPN.Core"}}

	srvrs.Config.API.IPs = NetConfigAPIInfoIPsArray{
		Type: "System.String[], mscorlib"}
	for _, val := range servers.Config.API.IPAddresses {
		srvrs.Config.API.IPs.Values = append(srvrs.Config.API.IPs.Values, val)
	}

	srvrs.OpenvpnServers = NetOpenvpnServerInfoArray{
		Type: "System.Collections.Generic.List`1[[IVPN.VpnProtocols.OpenVPN.OpenVPNVpnServer, IVPN.Core]], mscorlib"}
	srvrs.WireguardServers = NetWireGuardServerInfoArray{
		Type: "System.Collections.Generic.List`1[[IVPN.VpnProtocols.WireGuard.WireGuardVpnServerInfo, IVPN.Core]], mscorlib"}

	for _, val := range servers.OpenvpnServers {
		newSvr := NetOpenvpnServerInfo{
			Type: "IVPN.VpnProtocols.OpenVPN.OpenVPNVpnServer, IVPN.Core",

			Gateway:     val.Gateway,
			CountryCode: val.CountryCode,
			Country:     val.Country,
			City:        val.City,
			IPAddresses: IPAddressesArray{Type: "System.Collections.Generic.List`1[[System.String, mscorlib]], mscorlib"}}

		for _, addr := range val.IPAddresses {
			newSvr.IPAddresses.Values = append(newSvr.IPAddresses.Values, addr)
		}

		srvrs.OpenvpnServers.Values = append(srvrs.OpenvpnServers.Values, newSvr)
	}

	for _, val := range servers.WireguardServers {
		newSvr := NetWireGuardServerInfo{
			Type: "IVPN.VpnProtocols.WireGuard.WireGuardVpnServerInfo, IVPN.Core",

			Gateway:     val.Gateway,
			CountryCode: val.CountryCode,
			Country:     val.Country,
			City:        val.City,
			Hosts:       HostsArray{Type: "System.Collections.Generic.List`1[[IVPN.VpnProtocols.WireGuard.WireGuardVpnServerInfo+HostInfo, IVPN.Core]], mscorlib"}}

		for _, host := range val.Hosts {
			newSvr.Hosts.Values = append(newSvr.Hosts.Values, NetWireGuardServerHostInfo{
				Type: "IVPN.VpnProtocols.WireGuard.WireGuardVpnServerInfo+HostInfo, IVPN.Core",

				Hostname:  host.Hostname,
				Host:      host.Host,
				PublicKey: host.PublicKey,
				LocalIP:   host.LocalIP})
		}

		srvrs.WireguardServers.Values = append(srvrs.WireguardServers.Values, newSvr)
	}

	data, err := json.Marshal(IVPNServerListResponse{Type: "IVPN.IVPNServerListResponse, IVPN.Core", VpnServers: srvrs})
	if err != nil {
		log.Error("Error serializing response:", err)
		return nil
	}

	return data
}

// IVPNPingServersResponse returns average ping time for servers
func IVPNPingServersResponse(pingResult map[string]int) []byte {

	// EXAMPLE:
	//{
	//	"$type":"IVPN.IVPNPingServersResponse, IVPN.Core",
	//	"pingResults":{
	//	   "$type":"System.Collections.Generic.Dictionary`2[[System.String, mscorlib],[System.Int32, mscorlib]], mscorlib",
	//	   "127.0.0.1":77,
	//	   "127.0.0.2":88
	//	}
	// }

	type IVPNPingServersResponse struct {
		Type        string                 `json:"$type"`
		PingResults map[string]interface{} `json:"pingResults"`
	}

	results := make(map[string]interface{})
	results["$type"] = "System.Collections.Generic.Dictionary`2[[System.String, mscorlib],[System.Int32, mscorlib]], mscorlib"
	for k, v := range pingResult {
		results[k] = v
	}

	data, err := json.Marshal(IVPNPingServersResponse{Type: "IVPN.IVPNPingServersResponse, IVPN.Core", PingResults: results})
	if err != nil {
		log.Error("Error serializing response:", err)
		return nil
	}

	return data
}
