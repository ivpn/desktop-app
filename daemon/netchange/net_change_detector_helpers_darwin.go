//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2024 IVPN Limited.
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

package netchange

import (
	"fmt"
	"net"
	"strings"
	"syscall"

	"golang.org/x/net/route"
)

// Convert flags to human-readable strings
func flagsToString(flags int) string {
	var flagStrings []string
	if flags&syscall.RTF_UP != 0 {
		flagStrings = append(flagStrings, "UP")
	}
	if flags&syscall.RTF_GATEWAY != 0 {
		flagStrings = append(flagStrings, "GATEWAY")
	}
	if flags&syscall.RTF_HOST != 0 {
		flagStrings = append(flagStrings, "HOST")
	}
	if flags&syscall.RTF_REJECT != 0 {
		flagStrings = append(flagStrings, "REJECT")
	}
	if flags&syscall.RTF_DYNAMIC != 0 {
		flagStrings = append(flagStrings, "DYNAMIC")
	}
	if flags&syscall.RTF_MODIFIED != 0 {
		flagStrings = append(flagStrings, "MODIFIED")
	}
	if flags&syscall.RTF_DONE != 0 {
		flagStrings = append(flagStrings, "DONE")
	}
	if flags&syscall.RTF_DELCLONE != 0 {
		flagStrings = append(flagStrings, "DELCLONE")
	}
	if flags&syscall.RTF_CLONING != 0 {
		flagStrings = append(flagStrings, "CLONING")
	}
	if flags&syscall.RTF_XRESOLVE != 0 {
		flagStrings = append(flagStrings, "XRESOLVE")
	}
	if flags&syscall.RTF_LLINFO != 0 {
		flagStrings = append(flagStrings, "LLINFO")
	}
	if flags&syscall.RTF_STATIC != 0 {
		flagStrings = append(flagStrings, "STATIC")
	}
	if flags&syscall.RTF_BLACKHOLE != 0 {
		flagStrings = append(flagStrings, "BLACKHOLE")
	}
	if flags&syscall.RTF_PROTO1 != 0 {
		flagStrings = append(flagStrings, "PROTO1")
	}
	if flags&syscall.RTF_PROTO2 != 0 {
		flagStrings = append(flagStrings, "PROTO2")
	}
	if flags&syscall.RTF_PROTO3 != 0 {
		flagStrings = append(flagStrings, "PROTO3")
	}
	if flags&syscall.RTF_PINNED != 0 {
		flagStrings = append(flagStrings, "PINNED")
	}
	if flags&syscall.RTF_LOCAL != 0 {
		flagStrings = append(flagStrings, "LOCAL")
	}
	if flags&syscall.RTF_BROADCAST != 0 {
		flagStrings = append(flagStrings, "BROADCAST")
	}
	if flags&syscall.RTF_MULTICAST != 0 {
		flagStrings = append(flagStrings, "MULTICAST")
	}
	if flags&syscall.RTF_IFSCOPE != 0 {
		flagStrings = append(flagStrings, "IFSCOPE")
	}
	if flags&syscall.RTF_CONDEMNED != 0 {
		flagStrings = append(flagStrings, "CONDEMNED")
	}
	if flags&syscall.RTF_IFREF != 0 {
		flagStrings = append(flagStrings, "IFREF")
	}
	return strings.Join(flagStrings, "|")
}

func inet4AddrToString(addr *route.Inet4Addr) string {
	ip := net.IPv4(addr.IP[0], addr.IP[1], addr.IP[2], addr.IP[3])
	return ip.String()
}

func inet6AddrToString(addr *route.Inet6Addr) string {
	ip := net.IP(addr.IP[:])
	return ip.String()
}

func linkAddrToString(addr *route.LinkAddr) string {
	return fmt.Sprintf("LinkAddr: Index=%d, Name=%s", addr.Index, addr.Name)
}

func addrToString(addr route.Addr) string {
	switch a := addr.(type) {
	case *route.Inet4Addr:
		return fmt.Sprintf("Inet4Addr: %s", inet4AddrToString(a))
	case *route.Inet6Addr:
		return fmt.Sprintf("Inet6Addr: %s", inet6AddrToString(a))
	case *route.LinkAddr:
		return linkAddrToString(a)
	default:
		if addr != nil {
			return fmt.Sprintf("FAMILY %d ADDR. NOT SUPPORTED", addr.Family())
		}
		return ""
	}
}

func typeToString(typ int) string {
	switch typ {
	case syscall.RTM_ADD:
		return "RTM_ADD"
	case syscall.RTM_DELETE:
		return "RTM_DELETE"
	case syscall.RTM_CHANGE:
		return "RTM_CHANGE"
	case syscall.RTM_GET:
		return "RTM_GET"
	case syscall.RTM_LOSING:
		return "RTM_LOSING"
	case syscall.RTM_REDIRECT:
		return "RTM_REDIRECT"
	case syscall.RTM_MISS:
		return "RTM_MISS"
	case syscall.RTM_LOCK:
		return "RTM_LOCK"
	case syscall.RTM_OLDADD:
		return "RTM_OLDADD"
	case syscall.RTM_OLDDEL:
		return "RTM_OLDDEL"
	case syscall.RTM_RESOLVE:
		return "RTM_RESOLVE"
	case syscall.RTM_NEWADDR:
		return "RTM_NEWADDR"
	case syscall.RTM_DELADDR:
		return "RTM_DELADDR"
	case syscall.RTM_IFINFO:
		return "RTM_IFINFO"
	case syscall.RTM_NEWMADDR:
		return "RTM_NEWMADDR"
	case syscall.RTM_DELMADDR:
		return "RTM_DELMADDR"
	default:
		return fmt.Sprintf("UNKNOWN_TYPE(%d)", typ)
	}
}

func netmaskToCIDR(addr route.Addr) string {
	switch a := addr.(type) {
	case *route.Inet4Addr:
		ip := net.IPv4(a.IP[0], a.IP[1], a.IP[2], a.IP[3])
		ones, _ := net.IPv4Mask(a.IP[0], a.IP[1], a.IP[2], a.IP[3]).Size()
		return fmt.Sprintf("%s/%d", ip.String(), ones)
	case *route.Inet6Addr:
		ip := net.IP(a.IP[:])
		ones, _ := net.IPMask(a.IP[:]).Size()
		return fmt.Sprintf("%s/%d", ip.String(), ones)
	default:
		return addrToString(addr)
	}
}

// Print Addrs in human-readable format
func printAddrs(addrs []route.Addr) {
	const (
		RTAX_DST     = 0  // destination sockaddr present
		RTAX_GATEWAY = 1  // gateway sockaddr present
		RTAX_NETMASK = 2  // netmask sockaddr present
		RTAX_IFP     = 3  // interface name sockaddr present
		RTAX_IFA     = 4  // interface addr sockaddr present
		RTAX_AUTHOR  = 5  // sockaddr for author of redirect
		RTAX_BRD     = 6  // for NEWADDR, broadcast or p-p dest
		RTAX_SRC     = 7  // source sockaddr present
		RTAX_SRCMASK = 8  // source netmask present
		RTAX_LABEL   = 9  // route label present
		RTAX_MAX     = 10 // size of array to allocate
	)

	if len(addrs) > RTAX_DST && addrs[RTAX_DST] != nil {
		fmt.Printf("    Destination: %s\n", addrToString(addrs[RTAX_DST]))
	}
	if len(addrs) > RTAX_GATEWAY && addrs[RTAX_GATEWAY] != nil {
		fmt.Printf("    Gateway: %s\n", addrToString(addrs[RTAX_GATEWAY]))
	}
	if len(addrs) > RTAX_NETMASK && addrs[RTAX_NETMASK] != nil {
		fmt.Printf("    Netmask: %s\n", netmaskToCIDR(addrs[RTAX_NETMASK]))
	}
	if len(addrs) > RTAX_IFP && addrs[RTAX_IFP] != nil {
		fmt.Printf("    Interface Name: %s\n", addrToString(addrs[RTAX_IFP]))
	}
	if len(addrs) > RTAX_IFA && addrs[RTAX_IFA] != nil {
		fmt.Printf("    Interface Addr: %s\n", addrToString(addrs[RTAX_IFA]))
	}
	if len(addrs) > RTAX_AUTHOR && addrs[RTAX_AUTHOR] != nil {
		fmt.Printf("    Author: %s\n", addrToString(addrs[RTAX_AUTHOR]))
	}
	if len(addrs) > RTAX_BRD && addrs[RTAX_BRD] != nil {
		fmt.Printf("    Broadcast: %s\n", addrToString(addrs[RTAX_BRD]))
	}
	if len(addrs) > RTAX_SRC && addrs[RTAX_SRC] != nil {
		fmt.Printf("    Source: %s\n", addrToString(addrs[RTAX_SRC]))
	}
	if len(addrs) > RTAX_SRCMASK && addrs[RTAX_SRCMASK] != nil {
		fmt.Printf("    Source Netmask: %s\n", addrToString(addrs[RTAX_SRCMASK]))
	}
	if len(addrs) > RTAX_LABEL && addrs[RTAX_LABEL] != nil {
		fmt.Printf("    Label: %s\n", addrToString(addrs[RTAX_LABEL]))
	}
}

// Print RouteMessage in human-readable format
func printRouteMessage(rmsg route.RouteMessage) {
	fmt.Printf("RouteMessage: Type: %s | Flags: %s | Index: %d | ID: %d | Seq: %d | Version: %d\n",
		typeToString(rmsg.Type),
		flagsToString(rmsg.Flags),
		rmsg.Index,
		rmsg.ID,
		rmsg.Seq,
		rmsg.Version)
	fmt.Printf("  Addrs:\n")
	printAddrs(rmsg.Addrs)
}
