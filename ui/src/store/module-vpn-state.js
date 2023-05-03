//
//  UI for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
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
  PauseStateEnum,
  DnsEncryption,
  NormalizedConfigPortObject,
  NormalizedConfigPortRangeObject,
  PortTypeEnum,
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
      IsCanPause: null, //(true/false)
      IsTCP: false,
    }*/,

    disconnectedInfo: {
      ReasonDescription: "",
    },

    pauseState: PauseStateEnum.Resumed,

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
      IsEnabled: false,
      IsCanGetAppIconForBinary: false,
      // Split-Tunnelling (SplitTunnelStatus)
      // IsEnabled bool                     - Is ST enabled
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

    dns: {
      DnsHost: "",
      Encryption: DnsEncryption.None,
      DohTemplate: "",
    },

    currentWiFiInfo: null, //{ SSID: "", IsInsecureNetwork: false },
    availableWiFiNetworks: null, // []{SSID: ""}

    // Servers hash object: serversHashed[gateway] = server
    serversHashed: {},
    servers: {
      wireguard: [],
      openvpn: [],
      config: { ports: { wireguard: null, openvpn: null } },
    },

    // true when servers pinging in progress
    isPingingServers: false,

    // Pings info: hostsPings[host] = latency
    hostsPings: {},

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
      if (cs == VpnStateEnum.DISCONNECTED)
        state.pauseState = PauseStateEnum.Resumed;
    },
    connectionInfo(state, ci) {
      state.connectionInfo = ci;
      if (ci != null) {
        state.connectionState = VpnStateEnum.CONNECTED;
        state.disconnectedInfo = null;
      }
    },
    disconnected(state, disconnectionReason) {
      state.disconnectedInfo = { ReasonDescription: disconnectionReason };
      state.connectionState = VpnStateEnum.DISCONNECTED;
      state.pauseState = PauseStateEnum.Resumed;
      state.connectionInfo = null;
    },
    pauseState(state, val) {
      state.pauseState = val;
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
    dns(state, dns) {
      state.dns = dns;
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
          return true;
        default:
          return false;
      }
    },
    isConnected: (state) => {
      return state.connectionState === VpnStateEnum.CONNECTED;
    },
    vpnStateText: (state) => {
      return enumValueName(VpnStateEnum, state.connectionState);
    },
    activeServers(state, getters, rootState) {
      return getActiveServers(state, rootState);
    },
    isAntitrackerEnabled: (state) => {
      return isAntitrackerActive(state);
    },
    isAntitrackerHardcoreEnabled: (state) => {
      return isAntitrackerHardcoreActive(state);
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

        let ports = state.servers.config.ports.wireguard;
        if (vpnType === VpnTypeEnum.OpenVPN)
          ports = state.servers.config.ports.openvpn;

        // add custom ports
        try {
          if (
            rootState.settings.customPorts &&
            rootState.settings.customPorts.length > 0
          ) {
            let customPorts = rootState.settings.customPorts;
            // Filter custom port for current VPN type:
            // - WG supports only UDP ports
            // - custom port have to be in a range of allowed ports for current protocol
            const ranges = getters.funcGetConnectionPortRanges(vpnType);
            customPorts = customPorts.filter((p) => {
              if (!p) return false;
              // avoid duplicated
              if (isPortExists(ports, p) === true) {
                return false;
              }
              // WG supports only UDP ports
              if (
                vpnType === VpnTypeEnum.WireGuard &&
                p.type != PortTypeEnum.UDP
              )
                return false;
              // custom port have to be in a range of allowed ports for current protocol
              return isPortInAllowedRanges(ranges, p);
            });

            ports = ports.concat(customPorts);
          }
        } catch (e) {
          console.error(e);
        }

        if (!ports) return [];

        // normalize ports from configuration
        ports = ports
          .map((p) => NormalizedConfigPortObject(p))
          .filter((p) => p != null);

        // For Obfsproxy: only TCP protocol is applicable.
        try {
          const isUseObfsproxy =
            rootState.settings.daemonSettings.ObfsproxyConfig.Version > 0;
          if (vpnType === VpnTypeEnum.OpenVPN && isUseObfsproxy === true)
            ports = ports.filter((p) => p.type === PortTypeEnum.TCP);
        } catch (e) {
          console.error(e);
        }

        // return
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

      // apply port selection
      if (ci.ServerPort && ci.IsTCP != undefined) {
        let newPort = NormalizedConfigPortObject({
          type: ci.IsTCP ? PortTypeEnum.TCP : PortTypeEnum.UDP,
          port: ci.ServerPort,
        });
        if (newPort) {
          // Get applicable ports for current configuration
          // It is important to read 'connectionPorts' only after vpnType, multihop, obfsproxy was updated!
          const ports = context.getters.connectionPorts;
          // Check if the port exists in applicable ports list
          if (!isPortExists(ports, newPort)) {
            const portRagnes = context.getters.portRanges;
            if (isPortInAllowedRanges(portRagnes, newPort)) {
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
              newPort = changedPort;
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

      // save last DNS state
      context.commit("dns", ci.ManualDNS);
      updateDnsSettings(context);

      // save Mtu state (for WireGuard connections)
      if (ci.VpnType === VpnTypeEnum.WireGuard && Number.isInteger(ci.Mtu)) {
        var mtu = ci.Mtu;
        if (mtu === 0) mtu = null;
        context.commit("settings/mtu", mtu, { root: true });
      }
    },
    pauseState(context, val) {
      context.commit("pauseState", val);

      if (val === PauseStateEnum.Resumed || val === PauseStateEnum.Resuming)
        context.dispatch("uiState/pauseConnectionTill", null, { root: true });
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
    dns(context, dns) {
      context.commit("dns", dns);
      // save current state to settings
      updateDnsSettings(context);
    },
    firewallState(context, val) {
      context.commit("firewallState", val);
      // save current state to settings
      updateFirewallSettings(context);
    },
  },
};

function updateDnsSettings(context) {
  // save current state to settings
  const isAntitracker = isAntitrackerActive(context.state);
  context.dispatch("settings/isAntitracker", isAntitracker, { root: true });

  if (isAntitracker === true) {
    const isAntitrackerHardcore = isAntitrackerHardcoreActive(context.state);
    context.dispatch("settings/isAntitrackerHardcore", isAntitrackerHardcore, {
      root: true,
    });
  }

  if (isAntitracker === false) {
    let currDnsState = context.state.dns;

    let isCustomDns = true;
    if (currDnsState == null || !currDnsState.DnsHost) isCustomDns = false;
    else
      context.dispatch("settings/dnsCustomCfg", currDnsState, { root: true });

    context.dispatch("settings/dnsIsCustom", isCustomDns, { root: true });
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

function isAntitrackerActive(state) {
  let dnsIP = state.dns.DnsHost;
  if (!dnsIP || state.dns.Encryption != DnsEncryption.None) return false;

  let atConfig = state.servers.config.antitracker;
  switch (dnsIP) {
    case atConfig.default.ip:
    case atConfig.hardcore.ip:
      return true;
    default:
  }
  return false;
}

function isAntitrackerHardcoreActive(state) {
  let dnsIP = state.dns.DnsHost;
  if (!dnsIP || state.dns.Encryption != DnsEncryption.None) return false;

  let atConfig = state.servers.config.antitracker;
  switch (dnsIP) {
    case atConfig.hardcore.ip:
      return true;
    default:
  }
  return false;
}

function isPortInAllowedRanges(availablePortRanges, portToFind) {
  portToFind = NormalizedConfigPortObject(portToFind);
  if (!portToFind || !availablePortRanges) return false;
  const found = availablePortRanges.find(
    (p) =>
      p.type === portToFind.type &&
      portToFind.port >= p.range.min &&
      portToFind.port <= p.range.max
  );
  if (found) return true;
  return false;
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
