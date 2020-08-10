import { enumValueName, isStrNullOrEmpty } from "../helpers/helpers";
import {
  VpnTypeEnum,
  VpnStateEnum,
  PingQuality,
  PauseStateEnum
} from "./types";

export default {
  namespaced: true,

  state: {
    connectionState: VpnStateEnum.DISCONNECTED,

    connectionInfo: null /*{
      VpnType: VpnTypeEnum.OpenVPN,
      ConnectedSince: new Date(),
      ClientIP: "",
      ServerIP: "",
      ExitServerID: "",
      ManualDNS: ""
    }*/,

    disconnectedInfo: {
      ReasonDescription: ""
    },

    pauseState: PauseStateEnum.Resumed,

    firewallState: null /*{
      IsEnabled: null,
      IsPersistent: null,
      IsAllowLAN: null,
      IsAllowMulticast: null
    }*/,

    dns: "",

    // Servers hash object: serversHashed[gateway] = server
    serversHashed: {},
    servers: { wireguard: [], openvpn: [], config: {} }

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

          ping: ??? // property added after receiving ping info from daemon
          pingQuality: ??? // PingQuality (Good, Moderate, Bad) - property calculated after receiving ping info from daemon

          hosts: [
            {
              hostname: "",
              host: "",
              public_key: "",
              local_ip: ""
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
          ping: ??? // property added after receiving ping info from daemon
          pingQuality: ??? // PingQuality (Good, Moderate, Bad) - property calculated after receiving ping info from daemon
          
          ip_addresses: [""]
        }
      ],
      config: {
        antitracker: {
          default: { ip: "", "multihop-ip": "" },
          hardcore: { ip: "", "multihop-ip": "" }
        },
        api: { ips: [""] }
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
      }
    },
    disconnected(state, disconnectionReason) {
      state.disconnectedInfo = { ReasonDescription: disconnectionReason };
      state.connectionState = VpnStateEnum.DISCONNECTED;
      state.connectionInfo = null;
    },
    pauseState(state, val) {
      state.pauseState = val;
    },
    servers(state, serversObj) {
      updateServers(state, serversObj);
    },
    serversPingStatus(state, pingResultArray) {
      updateServersPings(state, pingResultArray);
    },
    firewallState(state, obj) {
      state.firewallState = obj;
    },
    dns(state, dns) {
      state.dns = dns;
    }
  },

  getters: {
    isDisconnected: state => {
      return state.connectionState === VpnStateEnum.DISCONNECTED;
    },
    vpnStateText: state => {
      return enumValueName(VpnStateEnum, state.connectionState);
    },
    activeServers(state, getters, rootState) {
      return getActiveServers(state, rootState.settings.vpnType);
    },
    antitrackerIp(state, getters, rootState) {
      let atConfig = state.servers.config.antitracker;
      if (atConfig == null) return null;
      let atIPs = rootState.settings.isAntitrackerHardcore
        ? atConfig.hardcore
        : atConfig.default;
      if (atIPs == null) return null;
      return rootState.settings.isMultiHop ? atIPs["multihop-ip"] : atIPs.ip;
    },
    isAntitrackerEnabled: state => {
      return isAntitrackerActive(state);
    },
    isAntitrackerHardcoreEnabled: state => {
      return isAntitrackerHardcoreActive(state);
    }
  },

  // can be called from renderer
  actions: {
    connectionInfo(context, ci) {
      // save current connection info
      context.commit("connectionInfo", ci);

      // Received 'connected' state
      // Connection can be triggered outside (not by current application instance)
      // So, we should just update received data in settings (vpnType, multihop, entry\exit servers)
      // (no consistency checks should be performed)
      const isMultiHop = isStrNullOrEmpty(ci.ExitServerID) ? false : true;
      context.commit("settings/vpnType", ci.VpnType, { root: true });
      context.dispatch("settings/isMultiHop", isMultiHop, { root: true });
      // it is important to read 'activeServers' only after vpnType was updated!
      const servers = context.getters.activeServers;
      const entrySvr = findServerByIp(servers, ci.ServerIP);
      context.commit("settings/serverEntry", entrySvr, { root: true });
      if (!isStrNullOrEmpty(ci.ExitServerID)) {
        const exitSvr = findServerByExitId(servers, ci.ExitServerID);
        context.commit("settings/serverExit", exitSvr, { root: true });
      }
    },
    pauseState(context, val) {
      context.commit("pauseState", val);

      if (val === PauseStateEnum.Resumed || val === PauseStateEnum.Resuming)
        context.dispatch("uiState/pauseConnectionTill", null, { root: true });
    },
    servers(context, value) {
      context.commit("servers", value);
      // notify 'settings' module about updated servers list
      // (it is required to update selected servers, if necessary)
      context.dispatch("settings/updateSelectedServers", null, { root: true });
    },
    dns(context, dns) {
      context.commit("dns", dns);
      // save current state to settings
      const isAntitracker = isAntitrackerActive(context.state);
      context.dispatch("settings/isAntitracker", isAntitracker, { root: true });
    }
  }
};

function getActiveServers(state, vpnType) {
  if (vpnType === VpnTypeEnum.OpenVPN) {
    return state.servers.openvpn;
  }
  return state.servers.wireguard;
}

function findServerByIp(servers, ip) {
  for (let i = 0; i < servers.length; i++) {
    const srv = servers[i];

    if (srv.hosts != null) {
      // wireguard server
      for (let j = 0; j < srv.hosts.length; j++) {
        if (srv.hosts[j].host === ip) return srv;
      }
    } else if (srv.ip_addresses !== null) {
      // openvpn server
      if (srv.ip_addresses.includes(ip)) return srv;
    }
  }
  return null;
}

function findServerByExitId(servers, id) {
  for (let i = 0; i < servers.length; i++) {
    const srv = servers[i];
    if (srv.gateway == null) continue;
    if (id === srv.gateway.split(".")[0]) return srv;
  }
}

function updateServersPings(state, pings) {
  let minPing = -1;
  let maxPing = -1;

  // hash new ping result by host
  let hashedPings = {};
  for (let i = 0; i < pings.length; i++) {
    hashedPings[pings[i].Host] = pings[i].Ping;
    if (pings[i].Ping > maxPing) maxPing = pings[i].Ping;
    if (minPing < 0 || pings[i].Ping < minPing) minPing = pings[i].Ping;
  }

  const pingMinMaxDiff = maxPing - minPing;

  function getPingQuality(ping) {
    if (ping == null || pingMinMaxDiff <= 0) return null;
    let relativePing = (ping - minPing) / pingMinMaxDiff;
    if (relativePing <= 0.5) return PingQuality.Good;
    else if (relativePing <= 0.8) return PingQuality.Moderate;
    return PingQuality.Bad;
  }

  state.servers.wireguard.forEach(s => {
    for (let i = 0; i < s.hosts.length; i++) {
      let pingValFoHost = hashedPings[s.hosts[i].host];
      if (pingValFoHost != null) {
        s.ping = pingValFoHost;
        s.pingQuality = getPingQuality(s.ping);
        break;
      }
    }
  });

  state.servers.openvpn.forEach(s => {
    for (let i = 0; i < s.ip_addresses.length; i++) {
      let pingValFoHost = hashedPings[s.ip_addresses[i]];
      if (pingValFoHost != null) {
        s.ping = pingValFoHost;
        s.pingQuality = getPingQuality(s.ping);
        break;
      }
    }
  });
}

function updateServers(state, newServers) {
  if (newServers == null) return;

  // ensure all required properties are defined (even with empty values)
  let serversEmpty = {
    wireguard: [],
    openvpn: [],
    config: {
      antitracker: {
        default: {},
        hardcore: {}
      },
      api: { ips: [] }
    }
  };
  newServers = Object.assign(serversEmpty, newServers);

  // prepare hash for new servers (hash by gateway id)
  function initNewServersAndCreateHash(hashObj, servers) {
    let retObj = hashObj;
    if (retObj == null) retObj = {};
    for (let i = 0; i < servers.length; i++) {
      servers[i].ping = null; // initialize 'ping' field to support VUE reactivity for it
      servers[i].pingQuality = null;
      retObj[servers[i].gateway] = servers[i]; // hash
    }
    return retObj;
  }

  let hash = initNewServersAndCreateHash(null, newServers.wireguard);
  state.serversHashed = initNewServersAndCreateHash(hash, newServers.openvpn);

  // copy ping value from old objects
  function copySvrsDataFromOld(oldServers, newServersHashed) {
    for (let i = 0; i < oldServers.length; i++) {
      let oldSrv = oldServers[i];
      let newSrv = newServersHashed[oldSrv.gateway];
      if (newSrv == null) {
        continue;
      }
      newSrv.ping = oldSrv.ping;
      newSrv.pingQuality = oldSrv.pingQuality;
    }
  }
  copySvrsDataFromOld(state.servers.wireguard, state.serversHashed);
  copySvrsDataFromOld(state.servers.openvpn, state.serversHashed);

  // sort new servers (by country/city)
  function compare(a, b) {
    let ret = a.country_code.localeCompare(b.country_code);
    if (ret != 0) return ret;
    return a.city.localeCompare(b.city);
  }
  newServers.wireguard.sort(compare);
  newServers.openvpn.sort(compare);

  // save servers
  state.servers = newServers;
}

function isAntitrackerActive(state) {
  if (isStrNullOrEmpty(state.dns)) return false;
  let atConfig = state.servers.config.antitracker;
  switch (state.dns) {
    case atConfig.default.ip:
    case atConfig.hardcore.ip:
    case atConfig.default["multihop-ip"]:
    case atConfig.hardcore["multihop-ip"]:
      return true;
    default:
  }
  return false;
}

function isAntitrackerHardcoreActive(state) {
  if (isStrNullOrEmpty(state.dns)) return false;
  let atConfig = state.servers.config.antitracker;
  switch (state.dns) {
    case atConfig.hardcore.ip:
    case atConfig.hardcore["multihop-ip"]:
      return true;
    default:
  }
  return false;
}
