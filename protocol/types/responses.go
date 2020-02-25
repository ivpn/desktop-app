package types

import (
	"encoding/json"

	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/service/api"
	"github.com/ivpn/desktop-app-daemon/vpn"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("prttyp")
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
		Type         string
		ErrorMessage string
	}

	data, _ := marshalRespnse(IVPNErrorResponse{Type: "Error", ErrorMessage: err.Error()})
	return data
}

// IVPNEmptyResponse empty response on request
func IVPNEmptyResponse() []byte {
	type IVPNEmptyResponse struct {
		Type string
	}

	data, _ := marshalRespnse(IVPNEmptyResponse{Type: "Empty"})
	return data
}

// IVPNServiceExitingResponse service is going to exit response
func IVPNServiceExitingResponse() []byte {
	type IVPNServiceExitingResponse struct {
		Type string
	}

	data, _ := marshalRespnse(IVPNServiceExitingResponse{Type: "ServiceExiting"})
	return data
}

// IVPNHelloResponse response on initial request
func IVPNHelloResponse() []byte {
	type IVPNHelloResponse struct {
		Type    string
		Version string
	}

	data, _ := marshalRespnse(IVPNHelloResponse{Type: "Hello", Version: "1.0"})
	return data
}

// IVPNKillSwitchStatusResponse returns kill-switch status
func IVPNKillSwitchStatusResponse(isEnabled, isPersistent, isAllowLAN, isAllowMulticast bool) []byte {

	type IVPNKillSwitchStatusResponse struct {
		Type             string
		IsEnabled        bool
		IsPersistent     bool
		IsAllowLAN       bool
		IsAllowMulticast bool
	}

	data, _ := marshalRespnse(IVPNKillSwitchStatusResponse{Type: "KillSwitchStatus", IsEnabled: isEnabled, IsPersistent: isPersistent, IsAllowLAN: isAllowLAN, IsAllowMulticast: isAllowMulticast})
	return data
}

// IVPNKillSwitchGetStatusResponse returns kill-switch status
// TODO: command can be replaced by 'IVPNKillSwitchStatusResponse'
func IVPNKillSwitchGetStatusResponse(status bool) []byte {
	type IVPNKillSwitchGetStatusResponse struct {
		Type      string
		IsEnabled bool
	}

	data, _ := marshalRespnse(IVPNKillSwitchGetStatusResponse{Type: "KillSwitchGetStatus", IsEnabled: status})
	return data
}

// IVPNKillSwitchGetIsPestistentResponse returns kill-switch persistance status
func IVPNKillSwitchGetIsPestistentResponse(status bool) []byte {
	type IVPNKillSwitchGetIsPestistentResponse struct {
		Type         string
		IsPersistent bool
	}

	data, _ := marshalRespnse(IVPNKillSwitchGetIsPestistentResponse{Type: "KillSwitchGetIsPestistent", IsPersistent: status})
	return data
}

// IVPNDiagnosticsGeneratedResponse returns info from daemon logs
func IVPNDiagnosticsGeneratedResponse(servLog string, servLog0 string) []byte {
	type IVPNDiagnosticsGeneratedResponse struct {
		Type           string
		ServiceLog     string
		ServiceLog0    string
		OpenvpnLog     string
		OpenvpnLog0    string
		EnvironmentLog string
	}

	data, _ := marshalRespnse(IVPNDiagnosticsGeneratedResponse{Type: "DiagnosticsGenerated", ServiceLog: servLog, ServiceLog0: servLog0})
	return data
}

// IVPNSetAlternateDNSResponse returns status of changing DNS
func IVPNSetAlternateDNSResponse(isSuccess bool, newDNS string) []byte {
	type IVPNSetAlternateDNSResponse struct {
		Type       string
		IsSuccess  bool
		ChangedDNS string
	}

	data, _ := marshalRespnse(IVPNSetAlternateDNSResponse{Type: "SetAlternateDns", IsSuccess: isSuccess, ChangedDNS: newDNS})
	return data
}

// IVPNConnectedResponse notifying about established connection
func IVPNConnectedResponse(timeSecFrom1970 int64, clientIP string, serverIP string, vpnType vpn.Type) []byte {
	type IVPNConnectedResponse struct {
		Type            string
		VpnType         vpn.Type
		TimeSecFrom1970 int64
		ClientIP        string
		ServerIP        string
	}

	data, _ := marshalRespnse(IVPNConnectedResponse{Type: "Connected",
		TimeSecFrom1970: timeSecFrom1970,
		ClientIP:        clientIP,
		ServerIP:        serverIP,
		VpnType:         vpnType})
	return data
}

// IVPNDisconnectedResponse notifying about stopped connetion
func IVPNDisconnectedResponse(failure bool, authrnticationError bool, reasonDescription string) []byte {
	type IVPNDisconnectedResponse struct {
		Type              string
		Failure           bool
		Reason            int
		ReasonDescription string
	}

	reason := 0
	if authrnticationError == true {
		reason = 1
	}

	data, _ := marshalRespnse(IVPNDisconnectedResponse{Type: "Disconnected",
		Failure:           failure,
		Reason:            reason,
		ReasonDescription: reasonDescription})
	return data
}

// IVPNVpnStateResponse returns VPN connection state
func IVPNVpnStateResponse(state string, additionalInfo string) []byte {
	type IVPNStateResponse struct {
		Type                string
		State               string
		StateAdditionalInfo string
	}

	data, _ := marshalRespnse(IVPNStateResponse{Type: "VpnState",
		State:               state,
		StateAdditionalInfo: additionalInfo})
	return data
}

// IVPNServerListResponse returns list of servers
func IVPNServerListResponse(servers *api.ServersInfoResponse) []byte {
	type IVPNServerListResponse struct {
		Type       string
		VpnServers api.ServersInfoResponse
	}

	data, err := json.Marshal(IVPNServerListResponse{Type: "ServerList", VpnServers: *servers})
	if err != nil {
		log.Error("Error serializing response:", err)
		return nil
	}

	return data
}

// IVPNPingServersResponse returns average ping time for servers
func IVPNPingServersResponse(pingResult map[string]int) []byte {
	type PingResultType struct {
		Host string
		Ping int
	}

	type IVPNPingServersResponse struct {
		Type        string
		PingResults []PingResultType
	}

	var results []PingResultType
	for k, v := range pingResult {
		results = append(results, PingResultType{Host: k, Ping: v})
	}

	data, err := json.Marshal(IVPNPingServersResponse{Type: "PingServers", PingResults: results})
	if err != nil {
		log.Error("Error serializing response:", err)
		return nil
	}

	return data
}
