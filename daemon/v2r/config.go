//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2021 Privatus Limited.
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

func createConfigFromTemplate(vmessIp string, vmessPort int, DokodemoIp string, DokodemoPort int, vnextUserId string) *V2RayConfig {
	jsonData := defaultConfigTemplate
	config := &V2RayConfig{}
	if err := json.Unmarshal([]byte(jsonData), config); err != nil {
		fmt.Println(err)
	}

	config.Inbounds[0].Settings.Address = DokodemoIp
	config.Inbounds[0].Settings.Port = DokodemoPort
	config.Outbounds[0].Settings.Vnext[0].Address = vmessIp
	config.Outbounds[0].Settings.Vnext[0].Port = vmessPort
	config.Outbounds[0].Settings.Vnext[0].Users[0].Id = vnextUserId

	return config
}

func CreateConfig_OutboundsQuick(vmessIp string, vmessPort int, DokodemoIp string, DokodemoPort int, vnextUserId string) *V2RayConfig {
	config := createConfigFromTemplate(vmessIp, vmessPort, DokodemoIp, DokodemoPort, vnextUserId)
	config.Outbounds[0].StreamSettings.Network = "quic"
	config.Outbounds[0].StreamSettings.TcpSettings = nil
	return config
}

func CreateConfig_OutboundsTcp(vmessIp string, vmessPort int, DokodemoIp string, DokodemoPort int, vnextUserId string) *V2RayConfig {
	config := createConfigFromTemplate(vmessIp, vmessPort, DokodemoIp, DokodemoPort, vnextUserId)
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
	if c.Inbounds[0].Settings.Address == "" {
		return fmt.Errorf("config.Inbounds[0].Settings.Address is empty")
	}
	if c.Inbounds[0].Settings.Port == 0 {
		return fmt.Errorf("config.Inbounds[0].Settings.Port is empty")
	}
	if c.Outbounds[0].Settings.Vnext[0].Address == "" {
		return fmt.Errorf("config.Outbounds[0].Settings.Vnext[0].Address is empty")
	}
	if c.Outbounds[0].Settings.Vnext[0].Port == 0 {
		return fmt.Errorf("config.Outbounds[0].Settings.Vnext[0].Port is empty")
	}
	if c.Outbounds[0].Settings.Vnext[0].Users[0].Id == "" {
		return fmt.Errorf("config.Outbounds[0].Settings.Vnext[0].Users[0].Id is empty")
	}
	return nil
}
