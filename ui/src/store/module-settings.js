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
import {
  VpnTypeEnum,
  PortTypeEnum,
  NormalizedConfigPortObject,
  ServersSortTypeEnum,
  ColorTheme,
  ColorThemeTrayIcon,
  DnsEncryption,
  V2RayObfuscationEnum,
  isPortInRanges,
} from "@/store/types";
import { enumValueName } from "@/helpers/helpers";
import { Platform, PlatformEnum } from "@/platform/platform";

const defaultPort = { port: 2049, type: PortTypeEnum.UDP };

const getDefaultState = () => {
  let defState = {
    // SettingsSessionUUID allows to detect situations when settings was erased
    // This value should be the same as on daemon side. If it differs - current settings should be erased to default state
    SettingsSessionUUID: "",

    // session
    isExpectedAccountToBeLoggedIn: false,

    // VPN
    enableIPv6InTunnel: false,
    vpnType: VpnTypeEnum.WireGuard,
    // MTU - According to Windows specification: "... For IPv4 the minimum value is 576 bytes. For IPv6 the minimum is value is 1280 bytes... "
    mtu: null, // (number: [1280-65535]) MTU option is applicable only for WireGuard connections
    isMultiHop: false,
    serverEntry: null,
    serverExit: null,
    serverEntryHostId: null, // property is defined when selected specific host (contains only host ID: "us-tx1.gw.ivpn.net" => "us-tx1")
    serverExitHostId: null, // property is defined when selected specific host (contains only host ID: "us-tx1.gw.ivpn.net" => "us-tx1")
    isFastestServer: true,
    isRandomServer: false,
    isRandomExitServer: false,

    openvpnObfsproxyConfig: {
      // 0 - do not use obfsproxy; obfs3 - 3; obfs4 - 4
      Version: 0,
      // Inter-Arrival Time (IAT). Applicable only for obfs4.
      //	The values of IAT-mode can be “0”, “1”, or “2” in obfs4
      //	0 -	means that the IAT-mode is disabled and that large packets will be split by the network drivers,
      //		whose network fingerprints could be detected by censors.
      //	1 - means splitting large packets into MTU-size packets instead of letting the network drivers do it.
      //		Here, the MTU is 1448 bytes for the Obfs4 Bridge. This means the smaller packets cannot be reassembled for analysis and censoring.
      //	2 - means splitting large packets into variable size packets. The sizes are defined in Obfs4.
      Obfs4Iat: 0,
    },

    V2RayConfig: {
      OpenVPN: V2RayObfuscationEnum.None,
      WireGuard: V2RayObfuscationEnum.None,
    },

    // Favorite gateway's list (strings [gatewayID of server]). Only gateway ID in use ("us-tx.wg.ivpn.net" => "us-tx")
    serversFavoriteList: [],
    //Favorite hosts list (strings [host.dns_name])
    hostsFavoriteListDnsNames: [],

    // List of servers to exclude from fastest servers list (gateway, strings)
    serversFastestExcludeList: [], // only gateway ID in use ("us-tx.wg.ivpn.net" => "us-tx")

    // general
    quitWithoutConfirmation: false,
    disconnectOnQuit: true,

    // This object received out FROM DAEMON!
    daemonSettings: {
      IsAutoconnectOnLaunch: false,
      IsAutoconnectOnLaunchDaemon: false,
      UserDefinedOvpnFile: "",
      IsLogging: false,

      WiFi: {
        // canApplyInBackground:
        //	false - means the daemon applies actions in background
        //	true - VPN connection and Firewall status can be changed ONLY when UI client is connected to the daemon (UI app is running)
        canApplyInBackground: false,

        connectVPNOnInsecureNetwork: false,

        trustedNetworksControl: true,
        defaultTrustStatusTrusted: null, // null/true/false
        networks: null, // []{ ssid: "" isTrusted: false }

        actions: {
          unTrustedConnectVpn: true,
          unTrustedEnableFirewall: true,

          trustedDisconnectVpn: true,
          trustedDisableFirewall: true,
        },
      },

      //UserPrefs: {
      //  // (Linux)
      //  Platform: {
      //    IsDnsMgmtOldStyle bool
      //  }
      //  // (macOS)
      //  Platform: { }
      //  // (Windows)
      //  Platform: { }
      //}
    },

    // connection

    port: {
      OpenVPN: defaultPort,
      WireGuard: defaultPort,
    },
    // portWireGuardBackup - the original port value for WireGuard (before it was changed to V2Ray-specific TCP port).
    // WireGuard port can be changed to V2Ray-specific TCP port if V2Ray obfuscation is enabled.
    // In this case, we need to remember original WireGuard port to be able to restore it back when V2Ray obfuscation is disabled.
    portWireGuardBackup: defaultPort,

    // custom ports defined by user (based on the applicable port range)
    customPorts: [], // [ {type: "UDP/TCP", port: "X", range: {min: X, max: X}}, ... ],

    ovpnProxyType: "",
    ovpnProxyServer: "",
    ovpnProxyPort: 0,
    ovpnProxyUser: "",
    ovpnProxyPass: "",

    // firewall
    firewallActivateOnConnect: true,
    firewallDeactivateOnDisconnect: true,

    antiTracker: {
      Enabled: false, //bool
      Hardcore: false, //bool
      AntiTrackerBlockListName: "", //string
    },

    // dns
    dnsIsCustom: false,
    dnsCustomCfg: {
      DnsHost: "",
      Encryption: DnsEncryption.None,
      DohTemplate: "",
    },

    // firewall
    firewallCfg: {
      userExceptions: "",
    },

    // Split-Tunnel
    splitTunnel: {
      // A list of applications which was selected from apps-list
      // and date of last usage for each app.
      // It allow us to sort app list in the order starting from last used apps.
      favoriteAppsList: [], // {AppBinaryPath, LastUsedDate}
    },

    // UI
    showGatewaysWithoutIPv6: true,
    minimizedUI: false,
    minimizeToTray: true,
    showAppInSystemDock: false,
    serversSortType: ServersSortTypeEnum.City,
    colorTheme: ColorTheme.system,
    colorThemeTrayIcon: ColorThemeTrayIcon.auto,
    connectSelectedMapLocation: false,
    windowRestorePosition: null, // {x=xxx, y=xxx}
    showHosts: false, //Enable selection of individual servers in server selection list

    showISPInfo: false, // "Show ISP info in servers list"
    multihopWarnSelectSameCountries: true, // "Warn me when selecting Multihop entry and exit servers located in the same country"
    multihopWarnSelectSameISPs: false, // "Warn me when selecting Multihop entry and exit servers operated by the same ISP"

    // updates
    updates: {
      isBetaProgram: false,
    },
    skipAppUpdate: {
      genericVersion: null,
      daemonVersion: null,
      uiVersion: null,
    },
  };

  if (Platform() === PlatformEnum.Linux) {
    // Not all Linux distro support tray icons.
    // Therefore, we have to change default config for Linux.
    defState.minimizeToTray = false;
    defState.showAppInSystemDock = true; // 'skip-taskbar' not applicable for Linux (since Electron v20)
  }

  return defState;
};

// initial state
let initialState = getDefaultState();

export default {
  namespaced: true,

  state: initialState,

  mutations: {
    replaceState(state, val) {
      Object.assign(state, val);
    },

    resetToDefaults(state) {
      Object.assign(state, getDefaultState());
    },

    deleteObsoletePropertiesAfterUpgrade(state) {
      if (state.hostsFavoriteList) {
        // Remove obsolete property after UPGRADE (from v3.10.0 or older)
        delete state.hostsFavoriteList;
      }
    },

    settingsSessionUUID(state, val) {
      state.SettingsSessionUUID = val;
    },

    isExpectedAccountToBeLoggedIn(state, val) {
      state.isExpectedAccountToBeLoggedIn = val;
    },

    enableIPv6InTunnel(state, val) {
      state.enableIPv6InTunnel = val;
    },
    vpnType(state, val) {
      state.vpnType = val;
    },
    mtu(state, val) {
      state.mtu = val;
    },
    isMultiHop(state, isMH) {
      state.isMultiHop = isMH;
    },
    serverEntry(state, srv) {
      if (srv == null || srv.gateway == null)
        throw new Error("Unable to change server. Wrong server object.");
      if (!isServerContainsHost(srv, state.serverEntryHostId))
        state.serverEntryHostId = null;
      state.serverEntry = srv;
    },
    serverExit(state, srv) {
      if (srv == null || srv.gateway == null)
        throw new Error("Unable to change server. Wrong server object.");
      if (!isServerContainsHost(srv, state.serverExitHostId))
        state.serverExitHostId = null;

      state.serverExit = srv;
    },
    serverEntryHostId(state, hostId) {
      if (hostId) {
        hostId = hostId.split(".")[0]; // convert hostname to hostId (if necessary)
        if (!isServerContainsHost(state.serverEntry, hostId)) hostId = null;
      }
      state.serverEntryHostId = hostId;
    },
    serverExitHostId(state, hostId) {
      if (hostId) {
        hostId = hostId.split(".")[0]; // convert hostname to hostId (if necessary)
        if (!isServerContainsHost(state.serverExit, hostId)) hostId = null;
      }
      state.serverExitHostId = hostId;
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

    openvpnObfsproxyConfig(state, val) {
      state.openvpnObfsproxyConfig = val;
    },
    setV2RayConfig(state, v2rayType) {
      state.V2RayConfig[enumValueName(VpnTypeEnum, state.vpnType)] = v2rayType; // V2RayObfuscationEnum
    },

    // Favorite gateway's list (strings)
    serversFavoriteList(state, val) {
      state.serversFavoriteList = val;
    },
    hostsFavoriteListDnsNames(state, val) {
      state.hostsFavoriteListDnsNames = val;
    },
    serversFastestExcludeList(state, val) {
      state.serversFastestExcludeList = val;
    },

    // general
    disconnectOnQuit(state, val) {
      state.disconnectOnQuit = val;
    },
    quitWithoutConfirmation(state, val) {
      state.quitWithoutConfirmation = val;
    },
    daemonSettings(state, val) {
      state.daemonSettings = val;
    },

    // connection
    setPort(state, portVal) {
      if (!portVal) {
        console.log("Warning! setPort() unable to set port. Port not defined.");
        return;
      }

      let newPort = Object.assign({}, state.port);

      if (state.vpnType === VpnTypeEnum.WireGuard) newPort.WireGuard = portVal;
      else if (state.vpnType === VpnTypeEnum.OpenVPN) newPort.OpenVPN = portVal;
      else {
        console.log("Warning! setPort() unable to set port. Unknown VPN type.");
        return;
      }

      doSetPortLogic(state, newPort);
    },
    port(state, val) {
      doSetPortLogic(state, val);
    },

    customPorts(state, val) {
      if (!state || !val) return;
      state.customPorts = val;
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
    firewallActivateOnConnect(state, val) {
      state.firewallActivateOnConnect = val;
    },
    firewallDeactivateOnDisconnect(state, val) {
      state.firewallDeactivateOnDisconnect = val;
    },

    // antitracker
    antiTracker(state, val) {
      if (!val)
        val = {
          Enabled: false,
          Hardcore: false,
          AntiTrackerBlockListName: "",
        };

      state.antiTracker = val;
    },

    // dns
    dnsIsCustom(state, val) {
      if (!val) val = false;
      state.dnsIsCustom = val;
    },
    dnsCustomCfg(state, val) {
      if (val) state.dnsCustomCfg = val;
      else {
        state.dnsIsCustom = false;
        state.dnsCustomCfg = {
          DnsHost: "",
          Encryption: DnsEncryption.None,
          DohTemplate: "",
        };
      }
    },
    firewallCfg(state, val) {
      state.firewallCfg = val;
    },

    // SplitTunnel
    splitTunnel(state, val) {
      state.splitTunnel = val;
    },
    // UI
    showGatewaysWithoutIPv6(state, val) {
      state.showGatewaysWithoutIPv6 = val;
    },
    minimizedUI(state, val) {
      state.minimizedUI = val;
    },
    minimizeToTray(state, val) {
      state.minimizeToTray = val;
    },
    connectSelectedMapLocation(state, val) {
      state.connectSelectedMapLocation = val;
    },
    showAppInSystemDock(state, val) {
      state.showAppInSystemDock = val;
    },
    serversSortType(state, val) {
      state.serversSortType = val;
    },
    colorTheme(state, val) {
      state.colorTheme = val;
    },
    colorThemeTrayIcon(state, val) {
      state.colorThemeTrayIcon = val;
    },
    windowRestorePosition(state, val) {
      state.windowRestorePosition = val;
    },
    showHosts(state, val) {
      state.showHosts = val;
      // if disabled - erase info about currently selected hosts
      if (val !== true) {
        state.serverEntryHostId = null;
        state.serverExitHostId = null;
      }
    },

    showISPInfo(state, val) {
      state.showISPInfo = val;
    },
    multihopWarnSelectSameCountries(state, val) {
      state.multihopWarnSelectSameCountries = val;
    },
    multihopWarnSelectSameISPs(state, val) {
      state.multihopWarnSelectSameISPs = val;
    },

    // updates
    updates(state, val) {
      state.updates = val;
    },
    skipAppUpdate(state, val) {
      state.skipAppUpdate = val;
    },
  },

  getters: {
    vpnType: (state) => {
      return state.vpnType;
    },
    isFastestServer: (state) => {
      if (state.isMultiHop) return false;
      return state.isFastestServer;
    },
    isRandomServer: (state) => {
      return state.isRandomServer;
    },
    isRandomExitServer: (state) => {
      if (!state.isMultiHop) return false;
      return state.isRandomExitServer;
    },
    getPort: (state) => {
      return state.port[enumValueName(VpnTypeEnum, state.vpnType)];
    },

    isConnectionUseObfsproxy: (state) => {
      try {
        if (state.vpnType !== VpnTypeEnum.OpenVPN) return false;
        return state.openvpnObfsproxyConfig.Version > 0;
      } catch (e) {
        console.error(e);
      }
      return false;
    },

    getV2RayConfig: (state) => {
      return state.V2RayConfig[enumValueName(VpnTypeEnum, state.vpnType)]; // V2RayObfuscationEnum
    },

    favoriteServersAndHosts: (state, getters, rootState, rootGetters) => {
      let ret = [];

      // All favorite servers (for all protocols)
      let favSvrs = state.serversFavoriteList;
      // servers for current protocol
      let activeServers = rootGetters["vpnState/activeServers"];

      // SERVERS (locations): Get favorite servers for current protocol
      try {
        if (favSvrs.length > 0) {
          // Filter only servers from 'favorite servers' list
          ret = activeServers.filter(
            (s) => favSvrs.includes(s.gateway.split(".")[0]) // Converting gateway name to geteway ID (if necessary). Example: "nl.gw.ivpn.net" => "nl"
          );
        }
      } catch (e) {
        console.error("Failed to get Favorite servers: ", e);
      }

      if (state.showHosts !== true) return ret;

      // HOSTS
      try {
        // All favorite host dns-names (for all protocols)
        let fHostDnsNames = state.hostsFavoriteListDnsNames;
        if (fHostDnsNames.length > 0) {
          // Looking for host objects for current protocol
          for (const svr of activeServers) {
            // If the server has only one host - check if this server is in 'serversFavoriteList' or not.
            // If it is already in favorite list - skip this single host for the server
            // (If server and it's single host are in favorites - we showing only server to user)
            if (svr.hosts.length == 1) {
              let svrGwId = svr.gateway.split(".")[0];
              if (favSvrs.includes(svrGwId)) continue;
            }
            for (const host of svr.hosts) {
              if (fHostDnsNames.includes(host.dns_name)) {
                // Host object extended by all properties from parent server object +favHostParentServerObj +favHost
                // favorite Host info object: {server..., favHost{}, favHostParentServerObj{}}, ...}
                let favHostExInfo = Object.assign({}, svr); // copy all info about location... etc.
                favHostExInfo = Object.assign(favHostExInfo, host); // overwrite host-related properties (like ping info)
                favHostExInfo.gateway = svr.gateway + ":" + host.hostname; // to avoid duplicate keys in UI lists
                favHostExInfo.favHostParentServerObj = svr; // original parent server object
                favHostExInfo.favHost = host; // original host object

                ret.push(favHostExInfo);
              }
            }
          }
        }
      } catch (e) {
        console.error("Failed to get Favorite hosts: ", e);
      }

      return ret;
    },

    // Returns list of applications for launching in split-tunnel mode
    // Taking into account 'favorite apps' list and 'installed apps' list
    // Resulted list is sorted by 'LastUsedDate' and 'AppName'
    getAppsToSplitTunnel: (state, getters, rootState) => {
      let installedApps = JSON.parse(
        JSON.stringify(rootState.allInstalledApps)
      );

      let getFileName = function (appBinPath) {
        return appBinPath.split("\\").pop().split("/").pop();
      };
      let getFileFolder = function (appBinPath) {
        const fname = getFileName(appBinPath);
        return appBinPath.substring(0, appBinPath.length - fname.length);
      };

      let retApps = [];
      if (installedApps) retApps = Object.assign(retApps, installedApps);

      // Add extra parameter 'LastUsedDate' to each element in app list (if exists in settings.splitTunnel.favoriteAppsList)
      // And add apps from 'favorite list' which are not present in common app list
      if (state.splitTunnel && state.splitTunnel.favoriteAppsList) {
        const favApps = state.splitTunnel.favoriteAppsList;
        let favAppsHashed = {};
        favApps.forEach((app) => {
          favAppsHashed[app.AppBinaryPath] = Object.assign({}, app);
        });
        // add LastUsedDate info
        retApps.forEach(function (element, index, theArray) {
          const knownAppInfo = favAppsHashed[element.AppBinaryPath];
          if (!knownAppInfo) return;
          knownAppInfo.isManual = false;
          theArray[index].LastUsedDate = knownAppInfo.LastUsedDate;
        });
        // add apps from 'favorite list' which are not present in common app list
        for (const favAppBinaryPath in favAppsHashed) {
          const favApp = favAppsHashed[favAppBinaryPath];
          if (!favApp || favApp.isManual === false) continue;
          if (!favApp.AppName) {
            favApp.AppName = getFileName(favApp.AppBinaryPath);
            favApp.AppGroup = getFileFolder(favApp.AppBinaryPath);
          }
          retApps.push(favApp);
        }
      }

      // sort applications: LastUsedDate + appName
      retApps.sort(function (a, b) {
        if (b.LastUsedDate && a.LastUsedDate)
          return new Date(b.LastUsedDate) - new Date(a.LastUsedDate);
        if (b.LastUsedDate && !a.LastUsedDate) return 1;
        if (!b.LastUsedDate && a.LastUsedDate) return -1;
        let aName = a.AppName;
        let bName = b.AppName;
        if (!aName) aName = "";
        if (!bName) bName = "";
        return aName.localeCompare(bName);
      });

      return retApps;
    },
  },

  // can be called from renderer
  actions: {
    resetToDefaults(context) {
      context.commit("resetToDefaults");
      // Necessary to initialize selected VPN servers
      const denyMultihopServersFromSameCountry = true;
      updateSelectedServers(context, denyMultihopServersFromSameCountry);
    },

    daemonSettings(context, val) {
      context.commit("daemonSettings", val);
    },

    isExpectedAccountToBeLoggedIn(context, val) {
      context.commit("isExpectedAccountToBeLoggedIn", val);
    },

    enableIPv6InTunnel(context, val) {
      context.commit("enableIPv6InTunnel", val);
    },
    vpnType(context, val) {
      context.commit("vpnType", val);
      // selected servers should be of correct VPN type. Necessary to update them
      updateSelectedServers(context);
    },
    mtu(context, val) {
      context.commit("mtu", val);
    },
    isMultiHop(context, val) {
      if (context.rootGetters["account/isMultihopAllowed"] === false)
        context.commit("isMultiHop", false);
      else {
        const oldVal = this.state.settings.isMultiHop;
        context.commit("isMultiHop", val);

        // do not change port if MH value was not changed (otherwise, new connection request will be sent)
        if (val === true && oldVal !== val) {
          const applicablePorts =
            context.rootGetters["vpnState/connectionPorts"];
          if (applicablePorts && applicablePorts.length > 0)
            context.commit(
              "setPort",
              getDefaultPortFromList(
                applicablePorts,
                context.getters["getPort"]
              )
            );
        }
      }
    },
    serverEntry(context, srv) {
      context.commit("serverEntry", srv);
      updateSelectedServers(context); // just to be sure entry-  and exit- servers are from different countries
    },
    serverExit(context, srv) {
      context.commit("serverExit", srv);
      updateSelectedServers(context); // just to be sure entry-  and exit- servers are from different countries
    },
    serverEntryHostId(context, hostId) {
      context.commit("serverEntryHostId", hostId);
    },
    serverExitHostId(context, hostId) {
      context.commit("serverExitHostId", hostId);
    },

    // Ensure the selected hosts equals to connected one. Of not equals - erase host selection
    serverEntryHostIPCheckOrErase(context, hostIp) {
      if (!this.state.settings.serverEntryHostId) return;
      const svr = this.state.settings.serverEntry;
      const hostId = this.state.settings.serverEntryHostId;
      let hostConnected = getServerHostByIp(svr, hostIp);
      let hostSelected = getServerHostById(svr, hostId);

      if (
        !hostConnected ||
        !hostSelected ||
        hostConnected.hostname != hostSelected.hostname
      )
        context.commit("serverEntryHostId", null);
    },
    serverExitHostCheckOrErase(context, hostname) {
      if (!this.state.settings.serverExitHostId) return;
      if (!hostname) {
        context.commit("serverExitHostId", null);
        return;
      }
      let hostId = hostname.split(".")[0]; // convert hostname to hostId (if necessary)
      if (hostId != this.state.settings.serverExitHostId)
        context.commit("serverExitHostId", null);
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

    openvpnObfsproxyConfig(context, val) {
      context.commit("openvpnObfsproxyConfig", val);
      ensurePortsSelectedCorrectly(context);
    },

    setV2RayConfig(context, v2rayVal) {
      context.commit("setV2RayConfig", v2rayVal); // V2RayObfuscationEnum
      ensurePortsSelectedCorrectly(context);
    },

    // Favorite gateway's list (strings)
    serversFavoriteList(context, val) {
      val = val.map((gw) => gw.split(".")[0]); // only gateway ID in use ("us-tx.wg.ivpn.net" => "us-tx")

      // remove servers from the favorite list if they are not exist anymore
      try {
        if (context.rootState.vpnState) {
          const hashedSvrs = context.rootState.vpnState.serversHashed;
          const hashedSrvGws = Object.keys(hashedSvrs);
          const hashedServersGwIds = hashedSrvGws.map((gw) => gw.split(".")[0]); // all available gateway IDs for all protocols
          const hashedServersGwIdsSet = new Set(hashedServersGwIds); // remove duplicates
          val = val.filter((gwID) => hashedServersGwIdsSet.has(gwID));
        }
      } catch (e) {
        console.error(e);
      }

      context.commit("serversFavoriteList", val);
    },
    hostsFavoriteListDnsNames(context, val) {
      // remove hosts from the favorite list if they are not exist anymore
      try {
        if (context.rootState.vpnState) {
          let allHostsDnsNamesSet = new Set();
          const hashedSvrs = context.rootState.vpnState.serversHashed;
          for (const [, svr] of Object.entries(hashedSvrs)) {
            if (!svr || !svr.hosts) continue;
            for (const h of svr.hosts) {
              if (!h.dns_name) continue;
              allHostsDnsNamesSet.add(h.dns_name);
            }
          }
          val = val.filter((host_dns) => allHostsDnsNamesSet.has(host_dns));
        }
      } catch (e) {
        console.error(e);
      }

      context.commit("hostsFavoriteListDnsNames", val);
    },
    serversFastestExcludeList(context, val) {
      context.commit("serversFastestExcludeList", val);
    },

    // general
    disconnectOnQuit(context, val) {
      context.commit("disconnectOnQuit", val);
    },
    quitWithoutConfirmation(context, val) {
      context.commit("quitWithoutConfirmation", val);
    },

    // connection
    setPort(context, portVal) {
      context.commit("setPort", portVal);
    },
    notifyAccessiblePortsInfo(context, accessiblePorts) {
      updatePortAccordingToAccessibleInfo(context, accessiblePorts);
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

    addNewCustomPort(context, val) {
      console.log("New custom port:", val);
      doAddNewCustomPort(context, val);
    },

    // firewall
    firewallActivateOnConnect(context, val) {
      context.commit("firewallActivateOnConnect", val);
    },
    firewallDeactivateOnDisconnect(context, val) {
      context.commit("firewallDeactivateOnDisconnect", val);
    },

    // antitracker
    antiTracker(context, val) {
      context.commit("antiTracker", val);
    },

    // dns
    dnsIsCustom(context, val) {
      context.commit("dnsIsCustom", val);
    },
    dnsCustomCfg(context, val) {
      context.commit("dnsCustomCfg", val);
    },
    firewallCfg(context, val) {
      context.commit("firewallCfg", val);
    },

    // SplitTunnel
    saveAddedAppCounter(context, appBinaryPath) {
      if (!appBinaryPath) return;

      let favoriteAppsList = [];
      if (
        this.state.settings.splitTunnel &&
        this.state.settings.splitTunnel.favoriteAppsList
      ) {
        // max len of 'favorite list' - 10 elements
        favoriteAppsList =
          this.state.settings.splitTunnel.favoriteAppsList.slice(0, 9);
      }

      let isFound = false;
      favoriteAppsList.forEach(function (element, index, theArray) {
        if (!element || !element.AppBinaryPath) return;
        if (element.AppBinaryPath == appBinaryPath) {
          theArray[index] = {
            AppBinaryPath: element.AppBinaryPath,
            LastUsedDate: new Date(),
          };
          isFound = true;
        }
      });
      if (isFound !== true) {
        favoriteAppsList.push({
          AppBinaryPath: appBinaryPath,
          LastUsedDate: new Date(),
        });
      }

      favoriteAppsList.sort(function (a, b) {
        return new Date(b.LastUsedDate) - new Date(a.LastUsedDate);
      });

      // create new (updated) splitTunnel object
      let st = {};
      if (this.state.settings.splitTunnel) {
        st = Object.assign(st, this.state.settings.splitTunnel);
      }
      st.favoriteAppsList = favoriteAppsList;
      context.commit("splitTunnel", st);
    },

    // UI
    showGatewaysWithoutIPv6(context, val) {
      context.commit("showGatewaysWithoutIPv6", val);
    },
    minimizedUI(context, val) {
      context.commit("minimizedUI", val);
    },
    minimizeToTray(context, val) {
      context.commit("minimizeToTray", val);
    },
    connectSelectedMapLocation(context, val) {
      context.commit("connectSelectedMapLocation", val);
    },
    showAppInSystemDock(context, val) {
      context.commit("showAppInSystemDock", val);
    },
    serversSortType(context, val) {
      context.commit("serversSortType", val);
    },
    colorTheme(context, val) {
      context.commit("colorTheme", val);
    },
    colorThemeTrayIcon(context, val) {
      context.commit("colorThemeTrayIcon", val);
    },
    showHosts(context, val) {
      context.commit("showHosts", val);
    },

    showISPInfo(context, val) {
      context.commit("showISPInfo", val);
    },
    multihopWarnSelectSameCountries(context, val) {
      context.commit("multihopWarnSelectSameCountries", val);
    },
    multihopWarnSelectSameISPs(context, val) {
      context.commit("multihopWarnSelectSameISPs", val);
    },

    // UPDATES
    updates(context, val) {
      context.commit("updates", val);
    },
    skipAppUpdate(context, val) {
      context.commit("skipAppUpdate", val);
    },

    // HELPERS
    updateSelectedServers(context) {
      updateSelectedServers(context);
      doSettingsUpgradeAfterSvrsUpdateIfRequired(context);
    },
  },
};

function doSetPortLogic(state, val) {
  if (!val || !val.OpenVPN || !val.WireGuard) {
    {
      console.log("Warning! port: unable to change port. Bad object.");
      return;
    }
  }

  // Save last good UDP port for WireGuard (since WireGuard supports only UDP)
  // It can happen than WireGuard port was changed to TCP port (if V2Ray obfuscation is enabled)
  // Keeping last good UDP port allows us to restore it back when V2Ray obfuscation is disabled
  if (val.WireGuard.type === PortTypeEnum.UDP)
    state.portWireGuardBackup = val.WireGuard;

  state.port = val;
}

function updateSelectedServers(context, isDenyMultihopServersFromSameCountry) {
  // - define selected servers (if not initialized)
  // - update selected servers if VPN type changed (try to use vpnType-related servers from the same location [country\city])
  // - if multi-hop entry- and exit- servers are from same country -> use first default exit server from another country
  // - ensure if selected servers exists in a servers list and using latest data
  // - ensure if selected ports exists in a servers configuration
  // TODO: ensure IPv6 configuration
  if (!context || !context.rootGetters || !context.rootState) {
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
    if (!server) return null;
    if (!server.hosts) return null;

    for (let h of server.hosts) {
      if (h && h.public_key) return VpnTypeEnum.WireGuard;
      else return VpnTypeEnum.OpenVPN;
    }
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

  // entry and exit servers should not have same gateway
  if (
    serverEntry != null &&
    serverExit != null &&
    serverEntry.gateway === serverExit.gateway
  ) {
    if (state.isRandomServer) serverEntry = null;
    else serverExit = null;
  }

  if (isDenyMultihopServersFromSameCountry === true) {
    // entry and exit servers should be from different countries
    if (
      serverEntry != null &&
      serverExit != null &&
      serverEntry.country_code === serverExit.country_code
    ) {
      if (state.isRandomServer) serverEntry = null;
      else serverExit = null;
    }
  }

  //
  // init selected servers (if not initialized)
  //
  let cnt = servers.length;

  // entryServer
  let fallbackEntryServer = null;
  for (let i = 0; serverEntry == null && i < cnt; i++) {
    if (serverExit == null) serverEntry = servers[i];
    else {
      if (servers[i].country_code !== serverExit.country_code) {
        if (!fallbackEntryServer) fallbackEntryServer = servers[i];
        if (servers[i].gateway !== serverExit.gateway) serverEntry = servers[i];
      }
    }
  }
  if (serverEntry == null) serverEntry = fallbackEntryServer; // fallback to first applicable server
  if (serverEntry == null && cnt > 0) serverEntry = servers[0]; // fallback to first server in a list

  // exitServer
  let fallbackExitServer = null;
  for (let i = 0; serverExit == null && i < cnt; i++) {
    if (serverEntry == null) serverExit = servers[i];
    else {
      if (servers[i].country_code !== serverEntry.country_code) {
        if (!fallbackExitServer) fallbackExitServer = servers[i];
        if (servers[i].gateway !== serverEntry.gateway) serverExit = servers[i];
      }
    }
  }
  if (serverExit == null) serverExit = fallbackExitServer; // fallback to first applicable server
  if (serverExit == null && cnt > 0) serverExit = servers[cnt - 1]; // fallback to last server in a list

  //
  // Update selected servers (in necessary)
  //
  if (serverEntry !== state.serverEntry)
    context.commit("serverEntry", serverEntry);
  if (serverExit !== state.serverExit) context.commit("serverExit", serverExit);

  // update selected hosts (if necessary)
  let entryHost = state.serverEntryHostId;
  let exitHost = state.serverExitHostId;
  if (entryHost && !isServerContainsHost(state.serverEntry, entryHost))
    context.commit("serverEntryHostId", null);
  if (exitHost && !isServerContainsHost(state.serverExit, exitHost))
    context.commit("serverExitHostId", null);

  //
  // Ensure if selected ports exists in a servers configuration or port is selected correctly
  //
  ensurePortsSelectedCorrectly(context);
}

function doSettingsUpgradeAfterSvrsUpdateIfRequired(context) {
  try {
    const state = context.state;
    if (state.hostsFavoriteList) {
      // UPGRADING from OLD SETTINGS (from v3.10.0 and older)
      // Property does not exists anymore: 'hostsFavoriteList' # Favorite hosts list (strings [host.hostname])
      // instead we use new property: 'hostsFavoriteListDnsNames' # Favorite hosts list (strings [host.dns_name])
      console.log("Upgrading old-style settings 'favorite hosts'...");

      let hostsFavListDnsNamesSet = new Set();
      const hashedSvrs = context.rootState.vpnState.serversHashed;
      for (const [, svr] of Object.entries(hashedSvrs)) {
        if (!svr || !svr.hosts) continue;
        for (const h of svr.hosts) {
          if (state.hostsFavoriteList.includes(h.hostname))
            hostsFavListDnsNamesSet.add(h.dns_name);
        }
      }

      // save converted data
      context.dispatch(
        "hostsFavoriteListDnsNames",
        Array.from(hostsFavListDnsNamesSet)
      );
      // forget 'hostsFavoriteList' forever
      context.commit("deleteObsoletePropertiesAfterUpgrade", null);
    }
  } catch (e) {
    console.error(e);
  }
}

function updatePortAccordingToAccessibleInfo(context, accessiblePorts) {
  if (!accessiblePorts || accessiblePorts.length == 0) {
    return;
  }
  // do not change port info if we already logged-in
  if (context.rootGetters["account/isLoggedIn"] !== false) return;

  let portString = function (p) {
    p = NormalizedConfigPortObject(p);
    return `${p.port}:${p.type}`;
  };

  let accessiblePortsHashed = {};
  for (let p of accessiblePorts) {
    accessiblePortsHashed[portString(p)] = p;
  }

  // check if selected port is in list of accessible ports
  let port = context.getters["getPort"];
  if (accessiblePortsHashed[portString(port)]) {
    return; // selected port is accessinle. Nothing to change
  }

  // get list of applicable ports
  let applicablePorts = context.rootGetters["vpnState/connectionPorts"];

  let portToApply = null;
  // looking for applicable port which is accessible
  for (let p of applicablePorts) {
    if (accessiblePortsHashed[portString(p)]) {
      portToApply = p;
      break;
    }
  }

  if (portToApply) {
    // apply new port
    context.dispatch("setPort", portToApply);
  }
}

function isPortExists(ports, port) {
  for (const configPort of ports) {
    const p = NormalizedConfigPortObject(configPort);
    if (p && p.type === port.type && p.port === port.port) return true;
  }
  return false;
}

// Ensure if selected ports exists in a servers configuration or port is selected correctly
function ensurePortsSelectedCorrectly(ctx) {
  if (!ctx || !ctx.rootGetters || !ctx.rootState) {
    console.error("ensurePortsSelectedCorrectly: failed (context not defined)");
    return;
  }

  // if we still not received configuration info (servers.json) - do nothing
  if (ctx.rootGetters["vpnState/isConfigInitialized"] !== true) return;

  // clean custom ports which are not applicable anymore (range is not exists anymore)
  eraseNonAcceptableCustomPorts(ctx);

  const state = ctx.state;

  let portOvpn = TestSuitablePort(ctx, VpnTypeEnum.OpenVPN);
  let portWg = TestSuitablePort(ctx, VpnTypeEnum.WireGuard);

  let cPort = Object.assign({}, state.port);
  if (portOvpn) cPort.OpenVPN = portOvpn;
  if (portWg) cPort.WireGuard = portWg;
  if (portOvpn || portWg) {
    ctx.commit("port", cPort);
    console.log(
      "(ensurePortsSelectedCorrectly) Port was changed from:",
      state.port,
      "to:",
      cPort
    );
  }
}

// returns null - if port is ok; otherwise - port value which have to be applied
function TestSuitablePort(context, vpnType) {
  var currPort =
    vpnType === VpnTypeEnum.OpenVPN
      ? context.state.port.OpenVPN
      : context.state.port.WireGuard;

  const funcGetPorts = context.rootGetters["vpnState/funcGetConnectionPorts"];
  const applicablePorts = funcGetPorts(vpnType);
  // Check is port applicable (according to current settings)
  if (
    applicablePorts &&
    applicablePorts.length > 0 &&
    !isPortExists(applicablePorts, currPort)
  ) {
    console.log(`Selected port `, currPort, "not applicable for VPN ", vpnType);
    return GetDefaultPort(context, applicablePorts, vpnType);
  }
  return null;
}

function GetDefaultPort(context, ports, vpnType) {
  let defPort = null;

  if (vpnType === VpnTypeEnum.WireGuard) {
    let alternatePort = context.state.portWireGuardBackup;
    if (isPortExists(ports, alternatePort)) return alternatePort;
  }

  // get V2Ray type
  let v2rayType = context.state.V2RayConfig.WireGuard;
  if (vpnType === VpnTypeEnum.OpenVPN)
    v2rayType = context.state.V2RayConfig.OpenVPN;

  for (const configPort of ports) {
    const p = NormalizedConfigPortObject(configPort);
    if (p) {
      defPort = p;
      if (!v2rayType) {
        return p;
      } else {
        // (for V2Ray) Recommended alternate ports are: 443/UDP (QUIC) and 80/TCP (HTTP)
        if (v2rayType === V2RayObfuscationEnum.QUIC) {
          if (p.port == 443 && p.type == PortTypeEnum.UDP) return p;
        } else if (v2rayType === V2RayObfuscationEnum.TCP) {
          if (p.port == 80 && p.type == PortTypeEnum.TCP) return p;
        }
      }
    }
  }
  return defPort;
}

function doAddNewCustomPort(context, port) {
  port = NormalizedConfigPortObject(port);
  if (!context || !port) return;
  try {
    const getRanges =
      context.rootGetters["vpnState/funcGetConnectionPortRanges"];

    const state = context.state;
    const currVpnPortRanges = getRanges(state.vpnType);

    // if true - skip port type checks when validating new port
    // (it is required for WireGuard ports when V2Ray/TCP is in use)
    let isSkipCheckPortRangesType = false;
    if (
      state.vpnType == VpnTypeEnum.WireGuard &&
      context.state.V2RayConfig.WireGuard === V2RayObfuscationEnum.TCP
    ) {
      isSkipCheckPortRangesType = true;
    }

    // check if port is acceptable: check if port is in allowed ranges for current VPN type
    if (!isPortInRanges(port, currVpnPortRanges, isSkipCheckPortRangesType))
      return;

    // check if custom port already exists
    if (isPortExists(state.customPorts, port) === true) return;

    const clone = function (obj) {
      return JSON.parse(JSON.stringify(obj));
    };

    // update custom ports
    let newCustomPorts = clone(state.customPorts);
    newCustomPorts.push(port);
    context.commit("customPorts", newCustomPorts);

    // apply port
    context.dispatch("setPort", port);

    // ensure that the port selected correctly
    ensurePortsSelectedCorrectly(context);
  } catch (e) {
    console.error(e);
  }
}

function eraseNonAcceptableCustomPorts(context) {
  try {
    if (!context) return;
    // if we still not received configuration info (servers.json) - do nothing
    if (context.rootGetters["vpnState/isConfigInitialized"] !== true) return;

    const getRanges =
      context.rootGetters["vpnState/funcGetConnectionPortRanges"];

    const state = context.state;

    const rangesOvpn = getRanges(VpnTypeEnum.OpenVPN);
    const rangesWg = getRanges(VpnTypeEnum.WireGuard);

    let newCustomPorts = [];
    state.customPorts.forEach((p) => {
      if (isPortInRanges(p, rangesOvpn) || isPortInRanges(p, rangesWg))
        newCustomPorts.push(p);
    });

    if (newCustomPorts.length != state.customPorts.length) {
      console.log(
        "Warning! Removing custom ports that do not belong to new ranges!"
      );
      context.commit("customPorts", newCustomPorts);
    }
  } catch (e) {
    console.error(e);
  }
}

function isServerContainsHost(server, hostID) {
  // hostID: "us-tx1.gw.ivpn.net" => "us-tx1"
  if (!hostID) return false;
  for (const h of server.hosts) {
    if (h.hostname.startsWith(hostID + ".")) return true;
  }
  return false;
}

function getServerHostByIp(server, hostIP) {
  if (!hostIP) return null;
  for (const h of server.hosts) {
    if (h.host == hostIP) return h;
  }
  return null;
}

function getServerHostById(server, hostId) {
  if (!hostId) return null;
  hostId = hostId.split(".")[0]; // convert hostname to hostId (if necessary)
  for (const h of server.hosts) {
    if (h.hostname.startsWith(hostId + ".")) return h;
  }
  return null;
}

function getDefaultPortFromList(ports, curPort) {
  let isDefaultPortExists = false;
  for (const f of ports) {
    if (f.type == defaultPort.type && f.port == defaultPort.port)
      isDefaultPortExists = true;
    if (f.type == curPort.type && f.port == curPort.port) return curPort;
  }
  if (isDefaultPortExists) return defaultPort;
  return ports[0];
}
