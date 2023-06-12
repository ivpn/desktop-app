//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 Privatus Limited.
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

//go:build darwin
// +build darwin

package vpnhelpers

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/netinfo"
	"github.com/ivpn/desktop-app/daemon/shell"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("cloudr")
}

// CloudSyncRouteUpdater replaces the "0/1 <vpnsvr>" route with "default <vpnsvr>" to enable iCloud synchronization.
// It also ensures the restoration of the original routes (e.g., after VPN disconnection).
//
// Problem description:
//
//	macOS uses the 'default' route for iCloud synchronization (e.g., Safari bookmarks, etc.)
//	When VPN is enabled and the "0/1 <vpnsvr>" route is defined, synchronization does not work.
//	To remedy this, we should replace the "0/1 <vpnsvr>" route with "default <vpnsvr>".
//
// Importantly, since the OS can modify the default route itself, we must be careful with it (to ensure that there are no leaks):
// - The functionality (using 'default' instead of "0/1") must be enabled ONLY when the IVPN firewall is activated (and "Allow local LAN" is deactivated).
type CloudSyncRouteUpdater struct {
	mutex                sync.Mutex
	originalDefaultRoute netinfo.Route // original default route
	vpnHost              net.IP        // internal IP address of VPN host
}

// Activate() saves initial (original) information about default route and updates the default route:
//
//	The function must be executed:
//	- after VPN connected (only if Firewall enabled)
//	- after VPN resume (only if Firewall enabled)
//	- (VPN enabled) after Firewall enabled
//
//	Logic:
//
//	If active and vpnHost != u.vpnHost:
//		Deactivate() and continue
//
//	If there is no original (DEFAULT) route - do nothing (return)
//
//	If not active AND "0/1 <vpn_host_IP>" exists AND "default <vpn_host_IP>" does not exists in routing table:
//		Save original (DEFAULT) information about default route.
//	   `$ route -n add default <DEFAULT> -ifscope <DEFAULT_INTERFACE>`
//	   `$ route -n delete default <DEFAULT>`
//	   `$ route -n add default <vpn_host_IP>`
//	   `$ route -n delete -net 0/1 <vpn_host_IP>`
//
//	On success: save state as 'activated' (save 'vpnHost' and original default gateway)
func (u *CloudSyncRouteUpdater) Activate(vpnHost net.IP) error {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	return u.doActivate(vpnHost)
}

// UpdateIfNeeded() Checks default routes and fix them to required values (if needed)
//
//	The function must be executed:
//	- (VPN and Firewall enabled) on changes in routing table (e.g. OS chnaged 'default route')
//
//	Logic:
//
//	If there is no original (DEFAULT) route - do nothing (return)
//
//	If "default <vpn_host_IP>" does not exist (default route was chnaged by OS)
//		Save original (DEFAULT) information about default route.
//	   `$ route -n add default <DEFAULT> -ifscope <DEFAULT_INTERFACE>`
//	   `$ route -n delete default <DEFAULT>`
//	   `$ route -n add default <vpn_host_IP>`
func (u *CloudSyncRouteUpdater) UpdateIfNeeded() error {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	return u.doUpdateIfNeeded()
}

// Deactivate() updates the routes (applicable only if activated):
//
//	The function must be executed:
//	- before disconnection VPN
//	- before VPN pause
//	- before Firewall disabling
//
//	Logic:
//
//	   route -n add -net 0/1 <vpn_host_IP>
//	   route -n delete default <vpn_host_IP>
//	   route -n add default <DEFAULT>
//	   route -n delete default <DEFAULT> -ifscope <DEFAULT_INTERFACE>
//
//	Save state as 'deactivated' (forget 'vpnHost' and original default gateway)
func (u *CloudSyncRouteUpdater) Deactivate() error {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	return u.doDeactivate()
}

func (u *CloudSyncRouteUpdater) IsActive() bool {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	return u.isActive()
}

func (u *CloudSyncRouteUpdater) doActivate(vpnHost net.IP) error {
	if vpnHost == nil || vpnHost.IsUnspecified() {
		return fmt.Errorf("iCloudSyncRouteUpdater: VPN host not specified")
	}

	//	If active and vpnHost != u.vpnHost:	Deactivate() and continue
	if u.isActive() && !vpnHost.Equal(u.vpnHost) {
		if err := u.doDeactivate(); err != nil {
			return err
		}
	}

	// Getting info about default routes
	//	outeVpn01       		netinfo.Route // "0/1     <vpn_host_IP>"
	//	routeVpnDefault 		netinfo.Route // "default <vpn_host_IP>"
	//	routeDefault    		netinfo.Route // "default <DEFAULT>"
	//  routeDefaultIfscoped	netinfo.Route // "default <DEFAULT> -ifscope <DEFAULT_INF_NAME>"
	routeVpn01, routeVpnDefault, routeDefault, routeDefaultIfscoped, _ := getRoutes(vpnHost, "")

	//	If there is no original (DEFAULT) route - do nothing (return)
	if !routeDefault.IsSpecified() {
		return nil
	}

	if u.isActive() {
		return fmt.Errorf("iCloudSyncRouteUpdater already activated")
	}

	//if !u.isActive() {
	//	If not active AND "0/1 <vpn_host_IP>" exists AND "default <vpn_host_IP>" does not exists in routing table:
	if !routeVpn01.IsSpecified() || routeVpnDefault.IsSpecified() {
		return nil
	}
	//if routeVpn01.IsSpecified() && !routeVpnDefault.IsSpecified() {
	// Save original (DEFAULT) information about default route.
	u.originalDefaultRoute = routeDefault
	//	   `$ route -n add default <DEFAULT> -ifscope <DEFAULT_INTERFACE>`
	//	   `$ route -n delete default <DEFAULT>`
	//	   `$ route -n add default <vpn_host_IP>`
	//	   `$ route -n delete -net 0/1 <vpn_host_IP>`
	if !routeDefaultIfscoped.IsSpecified() {
		if err := shell.Exec(log, "/sbin/route", "-n", "add", "default", routeDefault.GatewayIP.String(), "-ifscope", routeDefault.InterfaceName); err != nil {
			return fmt.Errorf("iCloudSyncRouteUpdater error: %w", err)
		}
	}
	if err := shell.Exec(log, "/sbin/route", "-n", "delete", "default", routeDefault.GatewayIP.String()); err != nil {
		return fmt.Errorf("iCloudSyncRouteUpdater error: %w", err)
	}
	if err := shell.Exec(log, "/sbin/route", "-n", "add", "default", routeVpn01.GatewayIP.String()); err != nil {
		return fmt.Errorf("iCloudSyncRouteUpdater error: %w", err)
	}
	if err := shell.Exec(log, "/sbin/route", "-n", "delete", "-net", "0/1", routeVpn01.GatewayIP.String()); err != nil {
		return fmt.Errorf("iCloudSyncRouteUpdater error: %w", err)
	}

	//	TODO: If failed - restore original routing data (revert successfully executed commands), mark as deactivated and return error

	//	On success: save state as 'activated' (save 'vpnHost' and original default gateway)
	u.vpnHost = vpnHost

	return nil
}

func (u *CloudSyncRouteUpdater) doUpdateIfNeeded() error {
	if !u.isActive() {
		return nil
	}

	// Getting info about default routes
	//	outeVpn01       		netinfo.Route // "0/1     <vpn_host_IP>"
	//	routeVpnDefault 		netinfo.Route // "default <vpn_host_IP>"
	//	routeDefault    		netinfo.Route // "default <DEFAULT>"
	//  routeDefaultIfscoped	netinfo.Route // "default <DEFAULT> -ifscope <DEFAULT_INF_NAME>"
	_, routeVpnDefault, routeDefault, routeDefaultIfscoped, _ := getRoutes(u.vpnHost, "")

	//	If there is no original (DEFAULT) route - do nothing (return)
	if !routeDefault.IsSpecified() {
		return nil
	}

	if routeVpnDefault.IsSpecified() {
		return nil
	}

	//active AND "default <vpn_host_IP>" does not exists: default route was chnaged by OS
	log.Info(fmt.Sprintf("Default gateway changed: %s -> %s. Updating routes...", routeVpnDefault.GatewayIP.String(), routeDefault.GatewayIP.String()))

	// Save original (DEFAULT) information about default route.
	u.originalDefaultRoute = routeDefault

	//	   `$ route -n add default <DEFAULT> -ifscope <DEFAULT_INTERFACE>`
	//	   `$ route -n delete default <DEFAULT>`
	//	   `$ route -n add default <vpn_host_IP>`
	if !routeDefaultIfscoped.IsSpecified() {
		if err := shell.Exec(log, "/sbin/route", "-n", "add", "default", routeDefault.GatewayIP.String(), "-ifscope", routeDefault.InterfaceName); err != nil {
			log.Error(fmt.Errorf("iCloudSyncRouteUpdater error : %w", err))
		}
	}
	if !routeVpnDefault.IsSpecified() {
		if err := shell.Exec(log, "/sbin/route", "-n", "delete", "default", routeDefault.GatewayIP.String()); err != nil {
			return fmt.Errorf("iCloudSyncRouteUpdater error : %w", err)
		}
		if err := shell.Exec(log, "/sbin/route", "-n", "add", "default", u.vpnHost.String()); err != nil {
			return fmt.Errorf("iCloudSyncRouteUpdater error : %w", err)
		}
	}

	return nil
}

func (u *CloudSyncRouteUpdater) doDeactivate() error {
	//	If not active - do nothing
	if !u.isActive() {
		return nil
	}

	// Getting info about default routes
	//	outeVpn01       		netinfo.Route // "0/1     <vpn_host_IP>"
	//	routeVpnDefault 		netinfo.Route // "default <vpn_host_IP>"
	//	routeDefault    		netinfo.Route // "default <DEFAULT>"
	//  routeDefaultIfscoped	netinfo.Route // "default <DEFAULT> -ifscope <DEFAULT_INF_NAME>"
	routeVpn01, routeVpnDefault, routeDefault, routeDefaultIfscoped, _ := getRoutes(u.vpnHost, u.originalDefaultRoute.InterfaceName)

	/////////////////////////////////////////
	//	   route -n add -net 0/1 <vpn_host_IP>
	//	   route -n delete default <vpn_host_IP>
	//	   route -n add default <DEFAULT>
	//	   route -n delete default <DEFAULT> -ifscope <DEFAULT_INTERFACE>
	if !routeVpn01.IsSpecified() {
		if err := shell.Exec(log, "/sbin/route", "-n", "add", "-net", "0/1", u.vpnHost.String()); err != nil {
			return fmt.Errorf("iCloudSyncRouteUpdater error : %w", err)
		}
	}

	if routeVpnDefault.IsSpecified() {
		if err := shell.Exec(log, "/sbin/route", "-n", "delete", "default", u.vpnHost.String()); err != nil {
			return fmt.Errorf("iCloudSyncRouteUpdater error : %w", err)
		}
	}

	if !routeDefault.IsSpecified() {
		if err := shell.Exec(log, "/sbin/route", "-n", "add", "default", u.originalDefaultRoute.GatewayIP.String()); err != nil {
			return fmt.Errorf("iCloudSyncRouteUpdater error : %w", err)
		}
	}

	if routeDefaultIfscoped.IsSpecified() && routeDefaultIfscoped.InterfaceName == u.originalDefaultRoute.InterfaceName {
		if err := shell.Exec(log, "/sbin/route", "-n", "delete", "default", u.originalDefaultRoute.GatewayIP.String(), "-ifscope", u.originalDefaultRoute.InterfaceName); err != nil {
			log.Error(err)
		}
	}

	// Save state as 'deactivated' (forget 'vpnHost' and original default gateway)
	u.originalDefaultRoute = netinfo.Route{}
	u.vpnHost = net.IP{}

	return nil
}

func (u *CloudSyncRouteUpdater) isActive() bool {
	if u.originalDefaultRoute.GatewayIP == nil || u.vpnHost == nil || u.originalDefaultRoute.GatewayIP.IsUnspecified() || u.vpnHost.IsUnspecified() {
		return false
	}
	return true
}

// getRoutes() reads routing table info
// Returns:
//
//	routeVpn01      		netinfo.Route // "0/1     <vpn_host_IP>"
//	routeVpnDefault 		netinfo.Route // "default <vpn_host_IP>"
//	routeDefault    		netinfo.Route // "default <DEFAULT>" 								(does not have "I" in flags)
//	routeDefaultIfscoped    netinfo.Route // "default <DEFAULT> -ifscope <DEFAULT_IF_NAME>"		(have "I" in flags)
func getRoutes(vpnHost net.IP, defaultRouteInfName string) (routeVpn01, routeVpnDefault, routeDefault, routeDefaultIfscoped netinfo.Route, err error) {
	// Getting info about default routes
	routes, _ := netinfo.GetDefaultRoutes()
	if len(routes) == 0 {
		return routeVpn01, routeVpnDefault, routeDefault, routeDefaultIfscoped, nil
	}
	for _, r := range routes {
		dst := strings.ToLower(r.Destination)

		//If gateway is not VPN IP
		if !r.GatewayIP.Equal(vpnHost) {
			// If default route
			if dst == "default" {
				// if default route does not defined with '-ifscope <interface>' (does not have "I" in flags)
				if !strings.Contains(r.Flags, "I") {
					routeDefault = r // 'default' (original default route)
					if defaultRouteInfName == "" {
						defaultRouteInfName = routeDefault.InterfaceName
					}
				} else if defaultRouteInfName != "" && r.InterfaceName == defaultRouteInfName {
					routeDefaultIfscoped = r
				}
			}
			continue
		}

		if dst == "0/1" && !routeVpn01.IsSpecified() {
			routeVpn01 = r
		} else if dst == "default" && !routeVpnDefault.IsSpecified() {
			routeVpnDefault = r
		}
	}
	return routeVpn01, routeVpnDefault, routeDefault, routeDefaultIfscoped, nil
}
