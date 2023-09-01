//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 IVPN Limited.
//
//  This file is part of the Daemon for IVPN Client Desktop.
//
//  The Daemon for IVPN Client Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The Daemon for IVPN Client Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the Daemon for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
//

package preferences

type WiFiNetwork struct {
	SSID      string `json:"ssid"`
	IsTrusted bool   `json:"isTrusted"`
}

type WiFiParams struct {
	// CanApplyInBackground:
	//	false - means the daemon applies actions in background
	//	true - VPN connection and Firewall status can be changed ONLY when UI client is connected to the daemon (UI app is running)
	CanApplyInBackground bool `json:"canApplyInBackground"`

	ConnectVPNOnInsecureNetwork bool `json:"connectVPNOnInsecureNetwork"`

	TrustedNetworksControl    bool          `json:"trustedNetworksControl"`
	DefaultTrustStatusTrusted *bool         `json:"defaultTrustStatusTrusted"` // nil - no trust action
	Networks                  []WiFiNetwork `json:"networks"`

	Actions struct {
		UnTrustedConnectVpn     bool `json:"unTrustedConnectVpn"`
		UnTrustedEnableFirewall bool `json:"unTrustedEnableFirewall"`
		UnTrustedBlockLan       bool `json:"unTrustedBlockLan"`
		TrustedDisconnectVpn    bool `json:"trustedDisconnectVpn"`
		TrustedDisableFirewall  bool `json:"trustedDisableFirewall"`
	} `json:"actions"`
}

func WiFiParamsCreate() WiFiParams {
	p := WiFiParams{}
	p.Actions.UnTrustedConnectVpn = true
	p.Actions.UnTrustedEnableFirewall = true
	p.Actions.UnTrustedBlockLan = true
	p.Actions.TrustedDisconnectVpn = true
	p.Actions.TrustedDisableFirewall = true
	return p
}
