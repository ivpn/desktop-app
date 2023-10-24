//
//  UI for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 IVPN Limited.
//
//  This file is part of the UI for IVPN Client Desktop.
//
//  The UI for IVPN Client Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The UI for IVPN Client Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the UI for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
//

import { enumValueName, getDistanceFromLatLonInKm } from "../helpers/helpers";
import { IsServerSupportIPv6 } from "@/helpers/helpers_servers";
import {
  VpnTypeEnum,
  VpnStateEnum,
  NormalizedConfigPortObject,
  NormalizedConfigPortRangeObject,
  PortTypeEnum,
  V2RayObfuscationEnum,
  isPortInRanges,
} from "./types";

export default {
  namespaced: true,

  state: {
    connectionState: VpnStateEnum.DISCONNECTED,

    connectionInfo: null /*{
      VpnType: VpnTypeEnum.OpenVPN,
      ConnectedSince: new Date(),
      ClientIP: "",
      ClientIPv6: "",
      ServerIP: "",
      ServerPort: 0,
      ExitHostname: "",
      ManualDNS: {
        DnsHost: "",      // string // DNS host IP address
	      Encryption: 0,    // DnsEncryption [	EncryptionNone = 0,	EncryptionDnsOverTls = 1,	EncryptionDnsOverHttps = 2]
	      DohTemplate: "",  // string // DoH/DoT template URI (for Encryption = DnsOverHttps or Encryption = DnsOverTls)
      },      
      IsTCP:      false,
      Mtu:        int ,  // (for WireGuard connections)	 
      IsPaused:   bool,  // When "true" - the actual connection may be "disconnected" (depending on the platform and VPN protocol), but the daemon responds "connected"   
      PausedTill  string // pausedTill.Format(time.RFC3339)
    }*/,

    disconnectedInfo: {
      ReasonDescription: "",
    },

    firewallState: {
      IsEnabled: null,
      IsPersistent: null,
      IsAllowLAN: null,
      IsAllowMulticast: null,
      IsAllowApiServers: null,
      UserExceptions: "",
    },

    // The split-tunnelling configuration
    splitTunnelling: {
      IsEnabled: false, // is ST enabled
      IsInversed: false, // Inverse Split Tunnel (only 'splitted' apps use VPN tunnel)
      IsAnyDns: false, // (only for Inverse Split Tunnel) When false: Allow only DNS servers specified by the IVPN application
      IsAllowWhenNoVpn: false, // (only for Inverse Split Tunnel) When true: Allow network access when VPN is not connected
      IsCanGetAppIconForBinary: false,
      // Split-Tunnelling (SplitTunnelStatus)
      // IsEnabled bool                     - Is ST enabled
      // IsInversed bool                    - Inversed split-tunneling (only 'splitted' apps use VPN tunnel)
      // IsCanGetAppIconForBinary bool      - This parameter informs availability of the functionality to get icon for particular binary
      //                                      (true - if commands GetAppIcon/AppIconResp  applicable for this platform)
      // SplitTunnelApps []string           - Information about applications added to ST configuration
      //                                      (applicable for Windows)
      // RunningApps []splittun.RunningApp  - Information about active applications running in Split-Tunnel environment
      //                                      (applicable for Linux)
      //                                      type RunningApp struct:
      //                                        Pid     int
      //                                        Ppid    int // The PID of the parent of this process.
      //                                        Cmdline string
      //                                        Exe         string  // The actual pathname of the executed command
      //                                        ExtIvpnRootPid int  // PID of the known parent process registered by AddPid() function
      //                                        ExtModifiedCmdLine string
    },

    currentWiFiInfo: null, //{ SSID: "", IsInsecureNetwork: false },
    availableWiFiNetworks: null, // []{SSID: ""}

    // true when servers pinging in progress
    isPingingServers: false,

    // Pings info: hostsPings[host] = latency
    hostsPings: {},

    // Servers hash object: serversHashed[gateway] = server
    serversHashed: {},
    servers: {
      wireguard: [],
      openvpn: [],
      config: { ports: { wireguard: null, openvpn: null } },
    },

    /*
    // SERVERS
    servers: {
      wireguard: [
        {
          gateway: "",
          country_code: "",
          country: "",
          city: "",
          latitude: 0,
          longitude: 0,
          isp: "",

          hosts: [
            {
              hostname: "",
              host: "",
              dns_name: "",
              public_key: "",
              local_ip: "",
              ipv6: 
              {                        
                local_ip: "",
                host: "",
                multihop_port: 0
              },
              load: 0.0,
            }
          ]
        }
      ],
      openvpn: [
        {
          gateway: "",
          country_code: "",
          country: "",
          city: "",
          latitude: 0,
          longitude: 0,
          isp: "",

          hosts: [
            {
              hostname: "",
              host: "",
              dns_name: "",
              multihop_port: 0,
              load: 0.0,
            }
          ]
        }
      ],
      config: {
        antitracker: {
          default: { ip: "" },
          hardcore: { ip: "" }
        },
        "antitracker_plus":{
          "DnsServers":[
            {
               "Name":"",
               "Description":"",
               "Normal":"", // IP string
               "Hardcore":"" // IP string
            },           
          ]
        },
        api: { ips: [""], ipv6s:[""] }
        ports: {
         openvpn: [ {type: "UDP/TCP", port: "X", range: {min: X, max: X}}, ... ],
         wireguard: [ {type: "UDP/TCP", port: "X", range: {min: X, max: X}}, ... ]
        }
      }
    }*/
  },

  mutations: {
    connectionState(state, cs) {
      state.connectionState = cs;
    },
    connectionInfo(state, ci) {
      state.connectionInfo = ci;
      if (ci != null) {
        state.connectionState = VpnStateEnum.CONNECTED;
        state.disconnectedInfo = null;

        // convert 'PausedTill' string to Date object
        let pausedTill = state.connectionInfo.PausedTill;
        if (pausedTill) state.connectionInfo.PausedTill = new Date(pausedTill);
      }
    },
    disconnected(state, disconnectionReason) {
      state.disconnectedInfo = { ReasonDescription: disconnectionReason };
      state.connectionState = VpnStateEnum.DISCONNECTED;
      state.connectionInfo = null;
    },
    setServersData(state, serversObj /*{servers,serversHashed}*/) {
      if (!serversObj || !serversObj.servers || !serversObj.serversHashed) {
        console.error("Unable to set servers data. Bad data object");
        return;
      }
      state.servers = serversObj.servers;
      state.serversHashed = serversObj.serversHashed;
    },
    isPingingServers(state, val) {
      state.isPingingServers = val;
    },
    hostsPings(state, val) {
      state.hostsPings = val;
    },
    firewallState(state, obj) {
      state.firewallState = obj;
    },
    // Split-Tunnelling
    splitTunnelling(state, val) {
      state.splitTunnelling = val;
    },

    currentWiFiInfo(state, currentWiFiInfo) {
      if (currentWiFiInfo != null && currentWiFiInfo.SSID == "")
        state.currentWiFiInfo = null;
      else state.currentWiFiInfo = currentWiFiInfo;
    },
    availableWiFiNetworks(state, availableWiFiNetworks) {
      state.availableWiFiNetworks = availableWiFiNetworks;
    },
  },

  getters: {
    isConfigInitialized: (state) => {
      if (
        !state ||
        !state.servers ||
        !state.servers.config ||
        !state.servers.config.ports
      )
        return false;
      let cfgPorts = state.servers.config.ports;
      if (!cfgPorts.openvpn || !cfgPorts.wireguard) return false;
      return true;
    },
    isDisconnecting: (state) => {
      return state.connectionState === VpnStateEnum.DISCONNECTING;
    },
    isDisconnected: (state) => {
      return state.connectionState === VpnStateEnum.DISCONNECTED;
    },
    isConnecting: (state) => {
      switch (state.connectionState) {
        case VpnStateEnum.CONNECTING:
        case VpnStateEnum.WAIT:
        case VpnStateEnum.AUTH:
        case VpnStateEnum.GETCONFIG:
        case VpnStateEnum.ASSIGNIP:
        case VpnStateEnum.ADDROUTES:
        case VpnStateEnum.RECONNECTING:
        case VpnStateEnum.TCP_CONNECT:
        case VpnStateEnum.INITIALISED:
          return true;
        default:
          return false;
      }
    },
    isConnected: (state) => {
      return state.connectionState === VpnStateEnum.CONNECTED;
    },
    isPaused: (state) => {
      if (!state.connectionInfo || !state.connectionInfo.IsPaused) return false;
      return state.connectionInfo.IsPaused;
    },
    isInverseSplitTunnel: (state) => {
      if (!state.splitTunnelling) return false;
      return (
        state.splitTunnelling.IsInversed && state.splitTunnelling.IsEnabled
      );
    },
    isInverseSplitTunnelAnyDns: (state, getters) => {
      return getters.isInverseSplitTunnel && state.splitTunnelling.IsAnyDns;
    },

    // IsAnyDns
    vpnStateText: (state) => {
      return enumValueName(VpnStateEnum, state.connectionState);
    },
    activeServers(state, getters, rootState) {
      return getActiveServers(state, rootState);
    },

    // fastestServer returns: the server with the lowest latency
    // (looking for the active servers that have latency info)
    // If there is no latency info for any server:
    // - return the nearest server (if geolocation info is known)
    // - else: return the currently selected server (if applicable)
    // - else: return the first server in the list (as a fallback)
    fastestServer(state, getters, rootState) {
      let servers = getActiveServers(state, rootState);
      if (servers == null || servers.length <= 0) return null;

      let skipSvrs = rootState.settings.serversFastestExcludeList;
      let retSvr = null;
      let retSvrPing = null;

      // If there will not be any server with ping-info -
      // save the info about the first applicable server (which is not in skipSvrs)
      let fallbackSvr = null;

      let getGatewayId = function (gatewayName) {
        return gatewayName.split(".")[0];
      };

      let selectedGwId = rootState.settings.serverEntry
        ? getGatewayId(rootState.settings.serverEntry.gateway)
        : null;

      const funcGetPing = getters["funcGetPing"];
      for (let i = 0; i < servers.length; i++) {
        let curSvr = servers[i];
        if (!curSvr) continue;

        // skip servers which user excluded from the 'fastest server' list
        const curGwID = getGatewayId(curSvr.gateway);
        if (skipSvrs.find((ss) => curGwID == getGatewayId(ss))) continue;

        if (!fallbackSvr && selectedGwId === curGwID) fallbackSvr = curSvr;

        const svrPing = funcGetPing(curSvr);
        if (
          svrPing &&
          svrPing > 0 &&
          (retSvr == null || retSvrPing > svrPing)
        ) {
          retSvr = curSvr;
          retSvrPing = svrPing;
        }
      }
      if (!fallbackSvr) fallbackSvr = servers[0];

      if (!retSvr) {
        // No fastest server detected (due to no ping info available)
        // Get nearest or first applicable server

        // get last known location
        const l = rootState.lastRealLocation;
        if (l) {
          try {
            // distance compare
            let compare = function (a, b) {
              var distA = getDistanceFromLatLonInKm(
                l.latitude,
                l.longitude,
                a.latitude,
                a.longitude
              );
              var distB = getDistanceFromLatLonInKm(
                l.latitude,
                l.longitude,
                b.latitude,
                b.longitude
              );
              if (distA === distB) return 0;
              if (distA < distB) return -1;
              return 1;
            };

            // sort servers by distance from last known real location
            let sortedSvrs = servers.slice().sort(compare);
            // get nearest server
            for (let i = 0; i < sortedSvrs.length; i++) {
              let curSvr = sortedSvrs[i];
              if (skipSvrs != null && skipSvrs.includes(curSvr.gateway))
                continue;
              retSvr = curSvr;
              break;
            }
          } catch (e) {
            console.log(e);
          }
        }

        // If still not found: choose the first applicable server
        if (!retSvr) retSvr = fallbackSvr;
      }

      return retSvr;
    },

    funcGetConnectionPorts: (state, getters, rootState) => (vpnType) => {
      try {
        if (vpnType == undefined || vpnType == null)
          vpnType = rootState.settings.vpnType;

        let customPortsType = PortTypeEnum.UDP; // type of applicable custom ports (null or PortTypeEnum.UDP/PortTypeEnum.TCP)
        let ports = state.servers.config.ports.wireguard;
        if (vpnType === VpnTypeEnum.OpenVPN) {
          customPortsType = null; // OpenVPN supports both UDP and TCP ports
          ports = state.servers.config.ports.openvpn;
        }
        if (!ports) ports = [];

        // normalize ports from configuration
        ports = ports
          .map((p) => NormalizedConfigPortObject(p))
          .filter((p) => p != null);

        // if true - skip port type checks when validating custom ports
        // (it is required for WireGuard ports when V2Ray/TCP is in use)
        let isSkipCheckPortRangesType = false;

        // --------------------------------------------------------
        // Update suitable ports according to current configuration
        // V2Ray (has precendance over Obfsproxy)
        // - V2Ray (TCP) uses only TCP ports
        // - V2Ray (QUIC) uses only UDP ports
        // Obfsproxy: only TCP protocol is applicable.
        // --------------------------------------------------------
        try {
          let v2rayType = rootState.settings.V2RayConfig.WireGuard;
          if (vpnType === VpnTypeEnum.OpenVPN)
            v2rayType = rootState.settings.V2RayConfig.OpenVPN;

          if (v2rayType === V2RayObfuscationEnum.QUIC) {
            // V2Ray (QUIC) uses only UDP ports
            customPortsType = PortTypeEnum.UDP;
            ports = ports.filter((p) => p.type === PortTypeEnum.UDP);
          } else if (v2rayType === V2RayObfuscationEnum.TCP) {
            // V2Ray (TCP) uses only TCP ports
            customPortsType = PortTypeEnum.TCP;
            const portsFiltered = ports.filter(
              (p) => p.type === PortTypeEnum.TCP
            );

            if (portsFiltered.length > 0) ports = portsFiltered;
            else if (vpnType === VpnTypeEnum.WireGuard) {
              // For WireGuard connection there will not be TCP ports defined. So we transform TCP ports to UDP ports
              isSkipCheckPortRangesType = true;
              ports = ports.map((p) => {
                return { port: p.port, type: PortTypeEnum.TCP };
              });
            }
          } else if (
            vpnType === VpnTypeEnum.OpenVPN &&
            rootState.settings.openvpnObfsproxyConfig.Version > 0
          ) {
            // For Obfsproxy: only TCP protocol is applicable.
            customPortsType = PortTypeEnum.TCP;
            ports = ports.filter((p) => p.type === PortTypeEnum.TCP);
          }
        } catch (e) {
          console.error(e);
        }

        // --------------------------------------------------------
        // Add custom ports (defined by user)
        // --------------------------------------------------------
        try {
          if (
            rootState.settings.customPorts &&
            rootState.settings.customPorts.length > 0
          ) {
            let customPorts = rootState.settings.customPorts;
            // Filter custom ports:
            // - skip duplicated
            // - port type have to be the same as customPortsType
            // - custom port have to be in a range of allowed ports for current protocol
            const ranges = getters.funcGetConnectionPortRanges(vpnType);
            customPorts = customPorts.filter((p) => {
              // avoid duplicated
              if (!p || isPortExists(ports, p) === true) {
                return false;
              }
              // filter custom ports by type
              if (customPortsType != null && p.type != customPortsType)
                return false;

              // custom port have to be in a range of allowed ports for current protocol
              return isPortInRanges(p, ranges, isSkipCheckPortRangesType);
            });

            ports = ports.concat(customPorts);
          }
        } catch (e) {
          console.error(e);
        }
        // --------------------------------------------------------
        return ports;
      } catch (e) {
        console.error(e);
        return [];
      }
    },

    connectionPorts(state, getters) {
      return getters.funcGetConnectionPorts();
    },

    funcGetConnectionPortRanges: (state, getters, rootState) => (vpnType) => {
      try {
        if (vpnType == undefined || vpnType == null)
          vpnType = rootState.settings.vpnType;

        let ports = state.servers.config.ports.wireguard;
        if (vpnType === VpnTypeEnum.OpenVPN)
          ports = state.servers.config.ports.openvpn;
        if (!ports) return [];

        let ranges = ports
          .map((p) => {
            return NormalizedConfigPortRangeObject(p);
          })
          .filter((p) => p != null);

        return ranges;
      } catch (e) {
        console.error(e);
        return [];
      }
    },

    portRanges(state, getters) {
      return getters.funcGetConnectionPortRanges();
    },

    funcGetPing: (state) => (hostOrLocation) => {
      try {
        if (!hostOrLocation) return null;
        if (hostOrLocation.hosts) {
          // server (location)
          const s = hostOrLocation;
          let best = null;
          for (const host of s.hosts) {
            let ping = state.hostsPings[host.host];
            if (!best || (ping && best > ping)) best = ping;
          }
          return best;
        }
        //host
        return state.hostsPings[hostOrLocation.host];
      } catch (e) {
        console.error(e);
        return null;
      }
    },
  },

  // can be called from renderer
  actions: {
    connectionInfo(context, ci) {
      // save current connection info
      context.commit("connectionInfo", ci);

      // Received 'connected' state
      // Connection can be triggered outside (not by current application instance)
      // So, we should just update received data in settings (vpnType, multihop, entry\exit servers and hosts)
      // (no consistency checks should be performed)
      const isMultiHop = !!ci.ExitHostname;
      context.commit("settings/vpnType", ci.VpnType, { root: true });
      context.dispatch("settings/isMultiHop", isMultiHop, { root: true });
      // it is important to read 'activeServers' only after vpnType was updated!
      const servers = context.getters.activeServers;
      const entrySvr = findServerByIp(servers, ci.ServerIP);
      context.commit("settings/serverEntry", entrySvr, { root: true });
      if (isMultiHop) {
        const exitSvr = findServerByHostname(servers, ci.ExitHostname);
        context.commit("settings/serverExit", exitSvr, { root: true });
      }
      // v2ray
      if (ci.V2RayProxy !== undefined) {
        context.dispatch("settings/setV2RayConfig", ci.V2RayProxy, {
          root: true,
        });
      }
      // obfsproxy
      if (ci.VpnType === VpnTypeEnum.OpenVPN && ci.Obfsproxy) {
        context.dispatch("settings/openvpnObfsproxyConfig", ci.Obfsproxy, {
          root: true,
        });
      }

      // apply port selection
      if (ci.ServerPort && ci.IsTCP != undefined) {
        let newPort = NormalizedConfigPortObject({
          type: ci.IsTCP ? PortTypeEnum.TCP : PortTypeEnum.UDP,
          port: ci.ServerPort,
        });
        if (newPort) {
          // Get applicable ports for current configuration
          // It is important to read 'connectionPorts' only after vpnType, V2RayType, multihop, obfsproxy was updated!
          const ports = context.getters.connectionPorts;
          // Check if the port exists in applicable ports list
          if (!isPortExists(ports, newPort)) {
            const portRagnes = context.getters.portRanges;
            if (isPortInRanges(newPort, portRagnes)) {
              // Outside connection (CLI) on custom port
              // Save new custom port into app settings
              context.dispatch("settings/addNewCustomPort", newPort, {
                root: true,
              });
            } else if (ports && ports.length > 0) {
              // New port does not exists. It could be because of multi-hop connection or/and obfsproxy.
              // (the port-base connection in use for MH or/and obfsproxy, so the final connection ports are not in a list)
              // For MH and obfsproxy only port type has sense. Looking for first applicable port

              let changedPort = null;
              // check if currently selected port fits to port type requirement
              // (do not change currently selected port if possible)
              let currPort = context.rootGetters["settings/getPort"];
              if (currPort && currPort.type === newPort.type)
                changedPort = currPort;

              // if nothing found - try to get first applicable port by type
              if (!changedPort)
                changedPort = ports.find((p) => p.type === newPort.type);
              if (changedPort) newPort = changedPort;
            }
          }
          context.commit("settings/setPort", newPort, { root: true });
        }
      }

      // Ensure the selected host equals to connected one. Of not equals - erase host selection
      context.dispatch("settings/serverEntryHostIPCheckOrErase", ci.ServerIP, {
        root: true,
      });
      context.dispatch("settings/serverExitHostCheckOrErase", ci.ExitHostname, {
        root: true,
      });

      // save current state to settings
      saveDnsSettings(context, ci.Dns);

      // save Mtu state (for WireGuard connections)
      if (ci.VpnType === VpnTypeEnum.WireGuard && Number.isInteger(ci.Mtu)) {
        var mtu = ci.Mtu;
        if (mtu === 0) mtu = null;
        context.commit("settings/mtu", mtu, { root: true });
      }
    },
    servers(context, value) {
      // Update servers data and hashes: {servers, serversHashed}
      // (avoid doing all calculations in mutation to do not freeze the UI!)
      const serversInfoObj = updateServers(context.state.servers, value);
      // Apply new servers data
      context.commit("setServersData", serversInfoObj);
      // notify 'settings' module about updated servers list
      // (it is required to update selected servers, selected ports ... etc. (if necessary))
      context.dispatch("settings/updateSelectedServers", null, { root: true });
    },
    updatePings(context, pings) {
      let hashedPings = {};
      for (let i = 0; i < pings.length; i++) {
        hashedPings[pings[i].Host] = pings[i].Ping;
      }
      context.commit("hostsPings", hashedPings);
    },
    // Split-Tunnelling
    splitTunnelling(state, val) {
      state.splitTunnelling = val;
    },
    dns(context, dnsStat) {
      // save current state to settings
      saveDnsSettings(context, dnsStat);
    },
    firewallState(context, val) {
      context.commit("firewallState", val);
      // save current state to settings
      updateFirewallSettings(context);
    },
  },
};

function saveDnsSettings(context, dnsStatus) {
  // dnsStatus:
  //{
  //  Dns: {
  //    DnsHost: "",
  //    Encryption: DnsEncryption.None,
  //    DohTemplate: "",
  //  },
  //  AntiTrackerStatus: {
  //    Enabled: false, //bool
  //    Hardcore: false, //bool
  //    AntiTrackerBlockListName: "", //string
  //  }
  //}

  if (!dnsStatus) {
    context.dispatch("settings/dnsCustomCfg", null, {
      root: true,
    });
    context.dispatch("settings/antiTracker", null, {
      root: true,
    });
    return;
  }

  // If AntiTracker is disabled - do not save empty AT settings (to keep current AntiTracker configuration. e.g. blocklist)
  context.dispatch("settings/antiTracker", dnsStatus.AntiTrackerStatus, {
    root: true,
  });

  if (dnsStatus.AntiTrackerStatus.Enabled !== true) {
    // Since AntiTracker has higher priority than custom DNS settings -
    // update custom DNS settings only if AntiTracker is disabled (custom DNS settings are in use)
    // This allows to keep custom DNS settings when AntiTracker is enabled/disabled
    let isCustomDns = !!dnsStatus.Dns && !!dnsStatus.Dns.DnsHost;

    context.dispatch("settings/dnsIsCustom", isCustomDns, {
      root: true,
    });

    if (isCustomDns) {
      context.dispatch("settings/dnsCustomCfg", dnsStatus.Dns, {
        root: true,
      });
    }
  }
}

function updateFirewallSettings(context) {
  // save current state to settings
  let firewallState = context.state.firewallState;
  var firewallCfg = { userExceptions: firewallState.UserExceptions };
  context.dispatch("settings/firewallCfg", firewallCfg, { root: true });
}

function getActiveServers(state, rootState) {
  const vpnType = rootState.settings.vpnType;
  const enableIPv6InTunnel = rootState.settings.enableIPv6InTunnel;
  const showGatewaysWithoutIPv6 = rootState.settings.showGatewaysWithoutIPv6;

  if (vpnType === VpnTypeEnum.OpenVPN) {
    // IPv6 in not implemented for OpenVPN
    return state.servers.openvpn;
  }

  let wgServers = state.servers.wireguard;
  if (enableIPv6InTunnel == true && showGatewaysWithoutIPv6 != true) {
    // show only servers which support IPv6
    return wgServers.filter((s) => {
      return IsServerSupportIPv6(s) === true;
    });
  }

  return wgServers;
}

function findServerByIp(servers, ip) {
  for (let i = 0; i < servers.length; i++) {
    const srv = servers[i];

    if (srv.hosts != null) {
      // wireguard/openvpn server
      for (let j = 0; j < srv.hosts.length; j++) {
        if (srv.hosts[j].host === ip) return srv;
      }
    }
  }
  return null;
}

function findServerByHostname(servers, hostname) {
  for (const srv of servers) {
    if (!srv || !srv.hosts) continue;
    for (const host of srv.hosts) {
      if (host.hostname == hostname) return srv;
    }
  }
}

function updateServers(oldServers, newServers) {
  if (newServers == null) return;

  // ensure all required properties are defined (even with empty values)
  let serversEmpty = {
    wireguard: [],
    openvpn: [],
    config: {
      antitracker: {
        default: {},
        hardcore: {},
      },
      api: { ips: [], ipv6s: [] },
    },
  };
  newServers = Object.assign(serversEmpty, newServers);

  // prepare hash for new servers (hash by gateway id)
  function initNewServersAndCreateHash(hashObj, servers) {
    let retObj = hashObj;
    if (retObj == null) retObj = {};
    for (let i = 0; i < servers.length; i++) {
      let svr = servers[i];
      retObj[svr.gateway] = svr; // hash
    }
    return retObj;
  }

  let hash = initNewServersAndCreateHash(null, newServers.wireguard);
  let serversHashed = initNewServersAndCreateHash(hash, newServers.openvpn);

  // sort new servers (by country/city)
  function compare(a, b) {
    let ret = a.country_code.localeCompare(b.country_code);
    if (ret != 0) return ret;
    return a.city.localeCompare(b.city);
  }
  newServers.wireguard.sort(compare);
  newServers.openvpn.sort(compare);

  return {
    servers: newServers,
    serversHashed: serversHashed,
  };
}

function isPortExists(availablePorts, portToFind) {
  portToFind = NormalizedConfigPortObject(portToFind);
  if (!portToFind || !availablePorts) return false;
  const found = availablePorts.find((p) => {
    p = NormalizedConfigPortObject(p);
    if (!p) return false;
    return p.type === portToFind.type && p.port === portToFind.port;
  });
  if (found) return true;
  return false;
}
