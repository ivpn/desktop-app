import store from "@/store";
import { VpnTypeEnum, DnsEncryption } from "@/store/types";

const ServerSelectionEnum = Object.freeze({
  Default: 0, // Server is manually defined
  Fastest: 1, // Fastest server in use (only for 'Entry' server)
  Random: 2, // Random server in use
});

export function InitConnectionParamsObject() {
  // Collecting the connection settings data

  let settings = store.state.settings;

  let manualDNS = {
    DnsHost: "",
    Encryption: DnsEncryption.None,
    DohTemplate: "",
  };

  const getHosts = function (server, customHostId) {
    if (!customHostId) return server.hosts;
    for (const h of server.hosts) {
      if (h.hostname.startsWith(customHostId + ".")) return [h];
    }
    return server.hosts;
  };

  let port = store.getters["settings/getPort"];

  let multihopExitSrvID = settings.isMultiHop
    ? settings.serverExit.gateway.split(".")[0]
    : "";

  let vpnParamsPropName = "";
  let vpnParamsObj = {};

  if (settings.vpnType === VpnTypeEnum.OpenVPN) {
    vpnParamsPropName = "OpenVpnParameters";

    vpnParamsObj = {
      EntryVpnServer: {
        Hosts: getHosts(settings.serverEntry, settings.serverEntryHostId),
      },

      MultihopExitServer: settings.isMultiHop
        ? {
            ExitSrvID: multihopExitSrvID,
            Hosts: getHosts(settings.serverExit, settings.serverExitHostId),
          }
        : null,

      Port: {
        Port: port.port,
        Protocol: port.type, // 0 === UDP
      },
    };

    const ProxyType = settings.ovpnProxyType;

    if (ProxyType && settings.ovpnProxyServer) {
      const ProxyPort = parseInt(settings.ovpnProxyPort);
      if (ProxyPort != null) {
        vpnParamsObj.Proxy = {
          Type: ProxyType,
          Address: settings.ovpnProxyServer,
          Port: ProxyPort,
          Username: settings.ovpnProxyUser,
          Password: settings.ovpnProxyPass,
        };
      }
    }
  } else {
    vpnParamsPropName = "WireGuardParameters";
    vpnParamsObj = {
      EntryVpnServer: {
        Hosts: getHosts(settings.serverEntry, settings.serverEntryHostId),
      },

      MultihopExitServer: settings.isMultiHop
        ? {
            ExitSrvID: multihopExitSrvID,
            Hosts: getHosts(settings.serverExit, settings.serverExitHostId),
          }
        : null,

      Port: {
        Port: port.port,
        Protocol: port.type, // 0 === UDP
      },
    };

    const mtu = Number.parseInt(settings.mtu);
    if (!Number.isNaN(mtu) && mtu >= 1280 && mtu <= 65535) {
      vpnParamsObj.Mtu = mtu;
    }
  }

  if (settings.dnsIsCustom) {
    manualDNS = settings.dnsCustomCfg;
  }

  // AntiTracker metadata
  let antiTrackerMetadata = { Enabled: false, Hardcore: false };
  if (settings.isAntitracker) {
    antiTrackerMetadata.Enabled = settings.isAntitracker;
    antiTrackerMetadata.Hardcore = settings.isAntitrackerHardcore;
  }

  // Metadata
  var metadata = {
    ServerSelectionEntry: ServerSelectionEnum.Default,
    ServerSelectionExit: ServerSelectionEnum.Default,
    AntiTracker: antiTrackerMetadata,
  };
  if (store.getters["settings/isFastestServer"]) {
    metadata.ServerSelectionEntry = ServerSelectionEnum.Fastest;
    metadata.FastestGatewaysExcludeList =
      store.state.settings.serversFastestExcludeList;
  } else if (store.getters["settings/isRandomServer"])
    metadata.ServerSelectionEntry = ServerSelectionEnum.Random;
  if (store.getters["settings/isRandomExitServer"])
    metadata.ServerSelectionExit = ServerSelectionEnum.Random;

  return {
    Metadata: metadata,
    VpnType: settings.vpnType,
    [vpnParamsPropName]: vpnParamsObj,
    ManualDNS: manualDNS,
    FirewallOn: store.state.settings.firewallActivateOnConnect === true,

    // Can use IPv6 connection inside tunnel
    // IPv6 has higher priority, if it supported by a server - we will use IPv6.
    // If IPv6 does not supported by server - we will use IPv4
    IPv6: settings.enableIPv6InTunnel,
    // Use ONLY IPv6 hosts (use IPv6 connection inside tunnel)
    // (ignored when IPv6!=true)
    IPv6Only: settings.showGatewaysWithoutIPv6 != true,
  };
}
