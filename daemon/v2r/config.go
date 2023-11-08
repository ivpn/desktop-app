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

package v2r

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const defaultConfigTemplate = `{
    "log": {
      "loglevel": "debug"
    },
    "inbounds": [
      {
        "port": "16661",
        "protocol":"dokodemo-door",
          "settings":{
            "address":"",
            "port":0,
            "network":"udp"
          }
      }
    ],
    "outbounds": [
      {
        "tag":"proxy",
        "protocol":"vmess",
        "settings":{
          "vnext":[
            {
              "address": "",
              "port": 0,
              "users":[
                {
                  "id": "",
                  "alterId":0,
                  "security": "none"
                }
              ]
            }
          ]
        },
        "streamSettings":{
          "network":"quic",
          "security":"tls",
          "quicSettings":{
            "security": "",
            "key": "",
            "header":{
              "type": "srtp"
            }
          },
          "tlsSettings":{
            "serverName":"xb1.gw.inet-telecom.com"
          },
          "tcpSettings": {
            "header": {
              "type": "http",
              "request": {
                "version": "1.1",
                "method": "GET",
                "path": ["/"],
                "headers": {
                  "Host": ["www.inet-telecom.com"],
                  "User-Agent": [
                    "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/55.0.2883.75 Safari/537.36",
                    "Mozilla/5.0 (iPhone; CPU iPhone OS 10_0_2 like Mac OS X) AppleWebKit/601.1 (KHTML, like Gecko) CriOS/53.0.2785.109 Mobile/14A456 Safari/601.1.46"
                  ],
                  "Accept-Encoding": ["gzip, deflate"],
                  "Connection": ["keep-alive"],
                  "Pragma": "no-cache"
                }
              }
            }
          }
        }
      }
    ]
  }`

// V2Ray configuration explanation
//
// V2Ray data flow:
//
//	[LOCAL-V2Ray-proxy] → (internet) →  [VMESS-server] → (IVPN-backend-infrastructure) → [VPN-server]
//
// Simplified V2Ray configuration:
//
//	{
//	    "inbounds": [
//	        {
//	            "port": "61554",                            // [LOCAL-V2Ray-proxy LOCAL-PORT]
//	            "settings": {
//	                "address": "169.150.252.110",           // [VPN-server IP]
//	                "port": 1443,                           // [VPN-server PORT]
//	                "network": "udp"                        // [VPN-server PROTOCOL]
//	            },
//	        }
//	    ],
//	    "outbounds": [
//	        {
//	            "settings": {
//	                "vnext": [
//	                    {
//	                        "address": "169.150.252.115",   // [VMESS-server IP]
//	                        "port": 2049,                   // [VMESS-server PORT]
//	                    }
//	                ]
//	            },
//	            "streamSettings": {
//	                "network": "tcp",                       // [ VMESS-PROTOCOL ]
//	            }
//	        }
//	    ]
//	}
//
// Fields explanation:
// * [VPN-server IP] - standard IP address of the VPN server.
// * [VPN-server PORT] and [VPN-server PROTOCOL] -  the port:protocol definition of the VPN server
//   - Multi-Hop: can be used any standard port:protocol applicable for VPN connection
//   - Single-Hop: (applicable V2Ray ports info available in servers.json under config->ports->v2ray)
//     -- WireGuard: can be 15351:UDP only
//     -- OpenVPN  : can be 20501,20502,20503,20504:UDP and 1443:TCP
//
// * [VMESS-server IP] - IP address of VMESS server, taken from v2ray field in host description in servers.json
// * [VMESS-server PORT] - PORT number of VMESS server. Can be ANY standard port from config->ports->openvpn/wireguard (limited only by [ VMESS-PROTOCOL ]):
//   - when VMESS/TCP is in use - Can be ANY standard TCP port (UDP ports not supported)
//   - when VMESS/QUIC is in use - Can be ANY standard UDP port (TCP ports not supported)
//
// * [ VMESS-PROTOCOL ] - protocol/obfuscation type
//   - quick for VMESS/QUICK
//   - tcp for VMESS/TCP
//
// Additional info:
// * V2Ray data flow:
//   - for Single-Hop: 	LocalV2RayProxy -> Outbound(EntryServer:V2Ray) -> Inbound(EntryServer:WireGuard)
//   - for Multi-Hop:	LocalV2RayProxy -> Outbound(EntryServer:V2Ray) -> Inbound(ExitServer:WireGuard)
//
// * Outbound ports can be any ports (which applicable for the selected VPN type).
//   - Preferred outbound ports: 80 for HTTP/VMess/TCP and 443 for HTTPS/VMess/QUIC
//
// * Inbound ports
//   - Single-Hop connections use internal V2Ray ports for inbound connections: svrs.Config.Ports.V2Ray
//   - Multi-Hop connections can use any ports for inbound connections (which applicable for the selected VPN type).
//
// Note!
// * Multi-Hop:
//   - For V2Ray connections we ignore port-based multihop configuration. Use default ports instead.
//   - WireGuard: since the first WG server is the ExitServer - we have to use it's public key in the WireGuard configuration
type V2RayConfig struct {
	Log struct {
		Loglevel string `json:"loglevel"`
	} `json:"log"`

	Inbounds []struct {
		Port     string `json:"port"` // Examples: "12345", "36373-57665"
		Protocol string `json:"protocol"`
		Settings struct {
			Address string `json:"address"`
			Port    int    `json:"port"`
			Network string `json:"network"` // Exmples: "tcp", "udp", "udp,tcp",
		} `json:"settings"`
	} `json:"inbounds"`

	Outbounds []struct {
		Tag      string `json:"tag"`
		Protocol string `json:"protocol"`
		Settings struct {
			Vnext []struct {
				Address string `json:"address"`
				Port    int    `json:"port"`
				Users   []struct {
					Id       string `json:"id"`
					AlterId  int    `json:"alterId"`
					Security string `json:"security"`
				} `json:"users"`
			} `json:"vnext"`
		} `json:"settings"`
		StreamSettings struct {
			Network  string `json:"network"`
			Security string `json:"security,omitempty"`

			QuicSettings *struct {
				Security string `json:"security"`
				Key      string `json:"key"`
				Header   struct {
					Type string `json:"type"`
				} `json:"header"`
			} `json:"quicSettings,omitempty"`

			TlsSettings *struct {
				ServerName string `json:"serverName"`
			} `json:"tlsSettings,omitempty"`

			TcpSettings *struct {
				Header struct {
					Type    string `json:"type"`
					Request struct {
						Version string   `json:"version"`
						Method  string   `json:"method"`
						Path    []string `json:"path"`
						Headers struct {
							Host           []string `json:"Host"`
							UserAgent      []string `json:"User-Agent"`
							AcceptEncoding []string `json:"Accept-Encoding"`
							Connection     []string `json:"Connection"`
							Pragma         string   `json:"Pragma"`
						} `json:"headers"`
					} `json:"request"`
				} `json:"header"`
			} `json:"tcpSettings,omitempty"`
		} `json:"streamSettings"`
	} `json:"outbounds"`
}

// GetLocalPort function returns local port and protocol
func (c *V2RayConfig) GetLocalPort() (port int, isTcp bool) {
	port, _ = strconv.Atoi(c.Inbounds[0].Port)
	isTcp = c.Inbounds[0].Settings.Network == "tcp"
	return
}

// SetLocalPort function sets local port and protocol
func (c *V2RayConfig) SetLocalPort(port int, isTcp bool) {
	c.Inbounds[0].Port = strconv.Itoa(port)
	if isTcp {
		c.Inbounds[0].Settings.Network = "tcp"
	} else {
		c.Inbounds[0].Settings.Network = "udp"
	}
}

func createConfigFromTemplate(outboundIp string, outboundPort int, inboundIp string, inboundPort int, outboundUserId string) *V2RayConfig {
	jsonData := defaultConfigTemplate
	config := &V2RayConfig{}
	if err := json.Unmarshal([]byte(jsonData), config); err != nil {
		fmt.Println(err)
	}

	config.Inbounds[0].Settings.Address = inboundIp
	config.Inbounds[0].Settings.Port = inboundPort
	config.Outbounds[0].Settings.Vnext[0].Address = outboundIp
	config.Outbounds[0].Settings.Vnext[0].Port = outboundPort
	config.Outbounds[0].Settings.Vnext[0].Users[0].Id = outboundUserId

	return config
}

func CreateConfig_OutboundsQuick(outboundIp string, outboundPort int, inboundIp string, inboundPort int, outboundUserId string, tlsSrvName string) *V2RayConfig {
	config := createConfigFromTemplate(outboundIp, outboundPort, inboundIp, inboundPort, outboundUserId)
	config.Outbounds[0].StreamSettings.Network = "quic"
	config.Outbounds[0].StreamSettings.TcpSettings = nil
	config.Outbounds[0].StreamSettings.TlsSettings.ServerName = tlsSrvName
	return config
}

func CreateConfig_OutboundsTcp(outboundIp string, outboundPort int, inboundIp string, inboundPort int, outboundUserId string) *V2RayConfig {
	config := createConfigFromTemplate(outboundIp, outboundPort, inboundIp, inboundPort, outboundUserId)
	config.Outbounds[0].StreamSettings.Network = "tcp"
	config.Outbounds[0].StreamSettings.Security = ""
	config.Outbounds[0].StreamSettings.QuicSettings = nil
	config.Outbounds[0].StreamSettings.TlsSettings = nil
	return config
}

// function checks if configuration fields of config are defined
func (c *V2RayConfig) isValid() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if port, _ := c.GetLocalPort(); port == 0 {
		return fmt.Errorf("config.Inbounds[0].Port or c.Inbounds[0].Settings.Network has invalid value")
	}
	if strings.TrimSpace(c.Inbounds[0].Settings.Address) == "" {
		return fmt.Errorf("config.Inbounds[0].Settings.Address is empty")
	}
	if c.Inbounds[0].Settings.Port == 0 {
		return fmt.Errorf("config.Inbounds[0].Settings.Port is empty")
	}
	if strings.TrimSpace(c.Outbounds[0].Settings.Vnext[0].Address) == "" {
		return fmt.Errorf("config.Outbounds[0].Settings.Vnext[0].Address is empty")
	}
	if c.Outbounds[0].Settings.Vnext[0].Port == 0 {
		return fmt.Errorf("config.Outbounds[0].Settings.Vnext[0].Port is empty")
	}
	if strings.TrimSpace(c.Outbounds[0].Settings.Vnext[0].Users[0].Id) == "" {
		return fmt.Errorf("config.Outbounds[0].Settings.Vnext[0].Users[0].Id is empty")
	}
	return nil
}
