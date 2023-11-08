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
	"errors"
	"fmt"

	"github.com/ivpn/desktop-app/daemon/netinfo"
)

// Start - helper function which starts V2Ray client with specified parameters
// It tryes to start V2Ray on the free port. In case of error it tryes to start V2Ray on another port (5 attemps)
// Note: To get local port it uses call V2RayWrapper.GetLocalPort()
// Parameters:
//
//	binary - path to V2Ray binary
//	tmpConfigFile - path to temporary config file
//	isTcpLocalPort - true if local port is must be TCP; false if UDP
//	outboundType - outbound type
//	outboundIp - IP address of VMess server
//	outboundPort - port of VMess server
//	inboundIp - IP address of Dokodemo server
//	inboundPort - port of Dokodemo server
//	vnextUserId - user ID
func Start(binary string,
	tmpConfigFile string,
	isTcpLocalPort bool,
	outboundType V2RayTransportType,
	outboundIp string,
	outboundPort int,
	inboundIp string,
	inboundPort int,
	outboundUserId string,
	quicTlsSvrName string) (*V2RayWrapper, error) {
	var cfg *V2RayConfig
	if outboundType == QUIC {
		if quicTlsSvrName == "" {
			return nil, errors.New("TLS server name is empty")
		}
		cfg = CreateConfig_OutboundsQuick(outboundIp, outboundPort, inboundIp, inboundPort, outboundUserId, quicTlsSvrName)
	} else if outboundType == TCP {
		cfg = CreateConfig_OutboundsTcp(outboundIp, outboundPort, inboundIp, inboundPort, outboundUserId)
	} else {
		return nil, errors.New("unknown outbound type")
	}

	var lastError error
	// Do 3 attemps to start v2ray with different ports (for the situation when port is already in use)
	for i := 0; i < 3; i++ {
		if i > 0 {
			log.Info("Retry to start V2Ray client with different port ...")
		}

		var (
			port int
			err  error
		)

		if isTcpLocalPort {
			port, err = netinfo.GetFreeTCPPort()
		} else {
			port, err = netinfo.GetFreeUDPPort()
		}
		if err != nil {
			log.Error(fmt.Sprintf("Failed to get free port: %v", err))
			continue
		}

		cfg.SetLocalPort(port, isTcpLocalPort)

		v := CreateV2RayWrapper(binary, tmpConfigFile, cfg)
		err = v.Start()
		if err != nil {
			lastError = err
			continue
		}

		return v, nil
	}
	return nil, lastError
}
