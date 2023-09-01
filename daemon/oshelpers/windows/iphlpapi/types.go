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

//go:build windows
// +build windows

package iphlpapi

// TCPTableClass - The TCP_TABLE_CLASS enumeration defines the set of values used to indicate the type of table returned by calls to GetExtendedTcpTable.
// https://docs.microsoft.com/en-us/windows/win32/api/iprtrmib/ne-iprtrmib-tcp_table_class
type TCPTableClass int

const (
	TCPTableBasicListener          TCPTableClass = iota
	TCPTableBasicConnections       TCPTableClass = iota
	TCPTableBasicAll               TCPTableClass = iota
	TCPTableOwnerPidListener       TCPTableClass = iota
	TCPTableOwnerPidConnections    TCPTableClass = iota
	TCPTableOwnerPidAll            TCPTableClass = iota
	TCPTableOwnerModuleListener    TCPTableClass = iota
	TCPTableOwnerModuleConnections TCPTableClass = iota
	TCPTableOwnerMuduleAll         TCPTableClass = iota
)

// APIMibIPForwardRow - MIB_IPFORWARDROW structure. https://docs.microsoft.com/ru-ru/windows/win32/api/ipmib/ns-ipmib-mib_ipforwardrow
type APIMibIPForwardRow struct {
	DwForwardDest      [4]byte
	DwForwardMask      [4]byte
	DwForwardPolicy    uint32
	DwForwardNextHop   [4]byte
	DwForwardIfIndex   uint32
	ForwardType        uint32
	ForwardProto       uint32
	DwForwardAge       uint32
	DwForwardNextHopAS uint32
	DwForwardMetric1   uint32
	DwForwardMetric2   uint32
	DwForwardMetric3   uint32
	DwForwardMetric4   uint32
	DwForwardMetric5   uint32
}

// MibTCPRowOwnerPid - The MIB_TCPROW_OWNER_PID structure contains information that describes an IPv4 TCP connection with IPv4 addresses, ports used by the TCP connection, and the specific process ID (PID) associated with connection.
// https://docs.microsoft.com/en-us/windows/win32/api/tcpmib/ns-tcpmib-mib_tcprow_owner_pid
type MibTCPRowOwnerPid struct {
	DwState      uint32
	DwLocalAddr  [4]byte //uint32
	DwLocalPort  [4]byte //uint32
	DwRemoteAddr [4]byte //uint32
	DwRemotePort [4]byte //uint32
	DwOwningPid  uint32
}
