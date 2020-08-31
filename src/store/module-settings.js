//
//  UI for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app-ui-beta
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

import { VpnTypeEnum, Ports, ServersSortTypeEnum } from "@/store/types";
import { enumValueName } from "@/helpers/helpers";
export default {
  namespaced: true,

  state: {
    vpnType: VpnTypeEnum.OpenVPN,
    isMultiHop: false,
    serverEntry: null,
    serverExit: null,
    isFastestServer: true,
    isRandomServer: false,
    isRandomExitServer: false,

    // Favorite gateway's list (strings)
    serversFavoriteList: [],
    // List of servers to exclude from fastest servers list (gateway, strings)
    serversFastestExcludeList: [],

    // general
    autoConnectOnLaunch: false,
    quitWithoutConfirmation: false,
    disconnectOnQuit: true,
    logging: false, // this parameter saves on the daemon's side

    // connection
    connectionUseObfsproxy: false, // this parameter saves on the daemon's side

    port: {
      OpenVPN: Ports.OpenVPN[0],
      WireGuard: Ports.WireGuard[0]
    },

    ovpnProxyType: "",
    ovpnProxyServer: "",
    ovpnProxyPort: 0,
    ovpnProxyUser: "",
    ovpnProxyPass: "",

    // firewall
    firewallOnOffOnConnect: true,
    firewallOffOnExit: true,

    // antitracker
    isAntitracker: false,
    isAntitrackerHardcore: false,

    // dns
    dnsIsCustom: false,
    dnsCustom: "",

    // UI
    minimizeToTray: true,
    showAppInSystemDock: false,
    serversSortType: ServersSortTypeEnum.City
  },

  mutations: {
    replaceState(state, val) {
      Object.assign(state, val);
    },

    vpnType(state, val) {
      state.vpnType = val;
      if (state.vpnType !== VpnTypeEnum.OpenVPN) state.isMultiHop = false;
    },
    isMultiHop(state, isMH) {
      if (state.vpnType === VpnTypeEnum.OpenVPN) {
        state.isMultiHop = isMH;
      } else state.isMultiHop = false;
    },
    serverEntry(state, srv) {
      if (srv == null || srv.gateway == null)
        throw new Error("Unable to change server. Wrong server object.");
      state.serverEntry = srv;
    },
    serverExit(state, srv) {
      if (srv == null || srv.gateway == null)
        throw new Error("Unable to change server. Wrong server object.");
      state.serverExit = srv;
    },
    isFastestServer(state, val) {
      state.isFastestServer = val;
      if (val === true) state.isRandomServer = false;
    },
    isRandomServer(state, val) {
      state.isRandomServer = val;
      if (val === true) state.isFastestServer = false;
    },
    isRandomExitServer(state, val) {
      state.isRandomExitServer = val;
    },

    // Favorite gateway's list (strings)
    serversFavoriteList(state, val) {
      state.serversFavoriteList = val;
    },
    serversFastestExcludeList(state, val) {
      state.serversFastestExcludeList = val;
    },

    // general
    autoConnectOnLaunch(state, val) {
      state.autoConnectOnLaunch = val;
    },
    disconnectOnQuit(state, val) {
      state.disconnectOnQuit = val;
    },
    quitWithoutConfirmation(state, val) {
      state.quitWithoutConfirmation = val;
    },
    logging(state, val) {
      state.logging = val;
    },

    // connection
    connectionUseObfsproxy(state, val) {
      state.connectionUseObfsproxy = val;
    },
    setPort(state, portVal) {
      state.port[enumValueName(VpnTypeEnum, state.vpnType)] = portVal;
    },

    ovpnProxyType(state, val) {
      state.ovpnProxyType = val;
    },
    ovpnProxyServer(state, val) {
      state.ovpnProxyServer = val;
    },
    ovpnProxyPort(state, val) {
      state.ovpnProxyPort = val;
    },
    ovpnProxyUser(state, val) {
      state.ovpnProxyUser = val;
    },
    ovpnProxyPass(state, val) {
      state.ovpnProxyPass = val;
    },

    // firewall
    firewallOnOffOnConnect(state, val) {
      state.firewallOnOffOnConnect = val;
    },
    firewallOffOnExit(state, val) {
      state.firewallOffOnExit = val;
    },

    // antitracker
    isAntitracker(state, val) {
      state.isAntitracker = val;
    },
    isAntitrackerHardcore(state, val) {
      state.isAntitrackerHardcore = val;
    },

    // dns
    dnsIsCustom(state, val) {
      state.dnsIsCustom = val;
    },
    dnsCustom(state, val) {
      state.dnsCustom = val;
    },

    //UI
    minimizeToTray(state, val) {
      state.minimizeToTray = val;
    },
    showAppInSystemDock(state, val) {
      state.showAppInSystemDock = val;
    },
    serversSortType(state, val) {
      state.serversSortType = val;
    }
  },

  getters: {
    vpnType: state => {
      return state.vpnType;
    },
    isFastestServer: state => {
      if (state.isMultiHop) return false;
      return state.isFastestServer;
    },
    isRandomServer: state => {
      return state.isRandomServer;
    },
    isRandomExitServer: state => {
      if (!state.isMultiHop) return false;
      return state.isRandomExitServer;
    },
    getPort: state => {
      return state.port[enumValueName(VpnTypeEnum, state.vpnType)];
    }
  },

  // can be called from renderer
  actions: {
    vpnType(context, val) {
      context.commit("vpnType", val);
      // selected servers should be of correct VPN type. Necessary to update them
      updateSelectedServers(context);
    },
    isMultiHop(context, val) {
      if (context.rootGetters["account/isMultihopAllowed"] === false)
        context.commit("isMultiHop", false);
      else context.commit("isMultiHop", val);
    },
    serverEntry(context, srv) {
      context.commit("serverEntry", srv);
      updateSelectedServers(context); // just to be sure entry-  and exit- servers are from different countries
    },
    serverExit(context, srv) {
      context.commit("serverExit", srv);
      updateSelectedServers(context); // just to be sure entry-  and exit- servers are from different countries
    },
    isFastestServer(context, val) {
      context.commit("isFastestServer", val);
    },
    isRandomServer(context, val) {
      context.commit("isRandomServer", val);
    },
    isRandomExitServer(context, val) {
      context.commit("isRandomExitServer", val);
    },

    // Favorite gateway's list (strings)
    serversFavoriteList(context, val) {
      context.commit("serversFavoriteList", val);
    },
    serversFastestExcludeList(context, val) {
      context.commit("serversFastestExcludeList", val);
    },

    // general
    autoConnectOnLaunch(context, val) {
      context.commit("autoConnectOnLaunch", val);
    },
    disconnectOnQuit(context, val) {
      context.commit("disconnectOnQuit", val);
    },
    quitWithoutConfirmation(context, val) {
      context.commit("quitWithoutConfirmation", val);
    },
    logging(context, val) {
      context.commit("logging", val);
    },

    // connection
    connectionUseObfsproxy(context, val) {
      context.commit("connectionUseObfsproxy", val);
    },
    setPort(context, portVal) {
      context.commit("setPort", portVal);
    },

    ovpnProxyType(context, val) {
      context.commit("ovpnProxyType", val);
    },
    ovpnProxyServer(context, val) {
      context.commit("ovpnProxyServer", val);
    },
    ovpnProxyPort(context, val) {
      context.commit("ovpnProxyPort", val);
    },
    ovpnProxyUser(context, val) {
      context.commit("ovpnProxyUser", val);
    },
    ovpnProxyPass(context, val) {
      context.commit("ovpnProxyPass", val);
    },

    // firewall
    firewallOnOffOnConnect(context, val) {
      context.commit("firewallOnOffOnConnect", val);
    },
    firewallOffOnExit(context, val) {
      context.commit("firewallOffOnExit", val);
    },

    // antitracker
    isAntitracker(context, val) {
      context.commit("isAntitracker", val);
    },
    isAntitrackerHardcore(context, val) {
      context.commit("isAntitrackerHardcore", val);
    },

    // dns
    dnsIsCustom(context, val) {
      context.commit("dnsIsCustom", val);
    },
    dnsCustom(context, val) {
      context.commit("dnsCustom", val);
    },

    //UI
    minimizeToTray(context, val) {
      context.commit("minimizeToTray", val);
    },
    showAppInSystemDock(context, val) {
      context.commit("showAppInSystemDock", val);
    },
    serversSortType(context, val) {
      context.commit("serversSortType", val);
    },

    // HELPERS
    updateSelectedServers(context) {
      updateSelectedServers(context);
    }
  }
};

function updateSelectedServers(context) {
  // - define selected servers (if not initialized)
  // - update selected servers if VPN type changed (try to use vpnType-related servers from the same location [country\city])
  // - if multi-hop entry- and exit- servers are from same country -> use first default exit server from another country
  // - ensure if selected servers exists in a servers list and using latest data

  if (
    context == null ||
    context.rootGetters == null ||
    context.rootState == null
  ) {
    console.error("Update selected servers failed (context not defined)");
    return;
  }

  const state = context.state;
  const servers = context.rootGetters["vpnState/activeServers"];
  const serversHashed = context.rootState.vpnState.serversHashed;
  if (servers.length <= 0) return;

  let serverEntry = state.serverEntry;
  let serverExit = state.serverExit;

  // HELPER FUNCTIONS
  function getVpnServerType(server) {
    if (server == null) return null;
    if (server.hosts != null) return VpnTypeEnum.WireGuard;
    if (server.ip_addresses != null) return VpnTypeEnum.OpenVPN;
    return null;
  }
  function findServerFromLocation(servers, countryCode, city) {
    let retServerByCountry = null;
    for (let i = 0; i < servers.length; i++) {
      let srv = servers[i];
      if (srv.country_code === countryCode) {
        if (srv.city === city) return srv;
        if (retServerByCountry != null) retServerByCountry = srv;
      }
    }
    return retServerByCountry;
  }

  // ensure if selected servers exists in a servers list and using latest data
  if (serverEntry != null) {
    serverEntry = serversHashed[serverEntry.gateway];
  }
  if (serverExit != null) {
    serverExit = serversHashed[serverExit.gateway];
  }

  // ensure selected servers have correct VPN type (if not - use correct server from same location)
  if (serverEntry != null) {
    if (getVpnServerType(serverEntry) !== state.vpnType) {
      serverEntry = findServerFromLocation(
        servers,
        serverEntry.country_code,
        serverEntry.city
      );
    }
  }
  if (serverExit != null) {
    if (getVpnServerType(serverExit) !== state.vpnType) {
      serverExit = findServerFromLocation(
        servers,
        serverExit.country_code,
        serverExit.city
      );
    }
  }
  // entry and exit servers should be from different countries
  if (
    serverEntry != null &&
    serverExit != null &&
    serverEntry.country_code === serverExit.country_code
  )
    serverExit = null;

  // init selected servers (if not initialized)
  let cnt = servers.length;
  for (let i = 0; serverEntry == null && i < cnt; i++) {
    if (
      serverExit == null ||
      servers[i].country_code !== serverExit.country_code
    ) {
      serverEntry = servers[i];
    }
  }
  for (let i = 0; serverExit == null && i < cnt; i++) {
    if (
      serverEntry == null ||
      servers[i].country_code !== serverEntry.country_code
    ) {
      serverExit = servers[i];
    }
  }

  if (serverEntry !== state.serverEntry)
    context.commit("serverEntry", serverEntry);
  if (serverExit !== state.serverExit) context.commit("serverExit", serverExit);
}
