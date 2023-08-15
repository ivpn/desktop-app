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

//go:build linux
// +build linux

package netlink

import (
	"fmt"
	"syscall"
)

// Listener provides possibility to listen for a netlink messages
//
// Usage example:
//
//	l, err := CreateListener()
//	if err != nil {
//		fmt.Println("Failed to initialize netlink listener: %s", err)
//		return
//	}
//	for {
//		msgs, err := l.ReadMsgs()
//		if err != nil {
//			fmt.Println("Could not read netlink messages: %s", err)
//		}
//		for _, m := range msgs {
//			if IsNewAddr(&m) || IsDelAddr(&m) {
//				fmt.Println("Address changed")
//			}
//		}
//	}
type Listener struct {
	fd int
	sa *syscall.SockaddrNetlink
}

// CreateListener creates new NetlinkListener object
func CreateListener() (*Listener, error) {
	groups := (1 << (syscall.RTNLGRP_LINK - 1)) |
		(1 << (syscall.RTNLGRP_IPV4_IFADDR - 1)) |
		(1 << (syscall.RTNLGRP_IPV6_IFADDR - 1))

	s, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_DGRAM,
		syscall.NETLINK_ROUTE)
	if err != nil {
		return nil, fmt.Errorf("socket initialization error: %s", err)
	}

	addr := &syscall.SockaddrNetlink{
		Family: syscall.AF_NETLINK,
		Pid:    uint32(0),
		Groups: uint32(groups),
	}

	err = syscall.Bind(s, addr)
	if err != nil {
		return nil, fmt.Errorf("socket binding error: %s", err)
	}

	return &Listener{fd: s, sa: addr}, nil
}

// ReadMsgs return received messages
func (l *Listener) ReadMsgs() ([]syscall.NetlinkMessage, error) {
	defer func() {
		recover()
	}()

	pkt := make([]byte, 4096)

	n, err := syscall.Read(l.fd, pkt)
	if err != nil {
		return nil, fmt.Errorf("NetlinkListener read error: %s", err)
	}

	msgs, err := syscall.ParseNetlinkMessage(pkt[:n])
	if err != nil {
		return nil, fmt.Errorf("NetlinkListener parse error: %s", err)
	}

	return msgs, nil
}

// IsNewAddr checking message type for syscall.RTM_NEWADDR
func IsNewAddr(msg *syscall.NetlinkMessage) bool {
	if msg.Header.Type == syscall.RTM_NEWADDR {
		return true
	}
	return false
}

// IsDelAddr checking message type for syscall.RTM_RTM_DELADDRNEWADDR
func IsDelAddr(msg *syscall.NetlinkMessage) bool {
	if msg.Header.Type == syscall.RTM_DELADDR {
		return true
	}
	return false
}
