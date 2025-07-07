import store from "@/store";
import { VpnTypeEnum, DnsEncryption } from "@/store/types";
import { resolveArrayConflicts } from "@/helpers/helpers";

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

      Obfs4proxy: settings.openvpnObfsproxyConfig,
      V2RayProxy: settings.V2RayConfig.OpenVPN,
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
      V2RayProxy: settings.V2RayConfig.WireGuard,
    };

    const mtu = Number.parseInt(settings.mtu);
    if (!Number.isNaN(mtu) && mtu >= 1280 && mtu <= 65535) {
      vpnParamsObj.Mtu = mtu;
    }
  }

  if (settings.dnsIsCustom) {
    manualDNS = settings.dnsCustomCfg;
  }

  // Metadata
  var metadata = {
    ServerSelectionEntry: ServerSelectionEnum.Default,
    ServerSelectionExit: ServerSelectionEnum.Default,
    AntiTracker: settings.antiTracker,
  };
  if (store.getters["settings/isFastestServer"]) {
    metadata.ServerSelectionEntry = ServerSelectionEnum.Fastest;
    metadata.FastestGatewaysExcludeList =
      store.state.settings.serversFastestExcludeList;
  } else if (store.getters["settings/isRandomServer"])
    metadata.ServerSelectionEntry = ServerSelectionEnum.Random;
  if (store.getters["settings/isRandomExitServer"])
    metadata.ServerSelectionExit = ServerSelectionEnum.Random;

  let fwOn = store.state.settings.firewallActivateOnConnect === true;

  // If multihopWarnSelectSameISPs is true, 
  // we need to ensure that service will not connect to hosts with the same ISP.
  if (settings.multihopWarnSelectSameISPs === true) {
    const entryHosts = vpnParamsObj?.EntryVpnServer?.Hosts;
    const exitHosts = vpnParamsObj?.MultihopExitServer?.Hosts;
   
    if (entryHosts && exitHosts) {
      const uniqueEntryISPs = [...new Set(entryHosts.map(host => host.isp).filter(isp => isp && isp.trim()))];
      const uniqueExitISPs = [...new Set(exitHosts.map(host => host.isp).filter(isp => isp && isp.trim()))];
      try {
        // Remove intersections between entry and exit ISPs
        // and return new arrays of ISPs for entry and exit servers
        const [newEntryISPs, newExitISPs] = resolveArrayConflicts(uniqueEntryISPs, uniqueExitISPs);

        vpnParamsObj.EntryVpnServer.Hosts = entryHosts.filter(host => newEntryISPs.includes(host.isp));
        vpnParamsObj.MultihopExitServer.Hosts = exitHosts.filter(host => newExitISPs.includes(host.isp));
      } catch (err) {
        // function resolveArrayConflicts can throw an error if conflicts cannot be resolved
        // in this case we will not change the hosts
        console.warn("Unable to resolve ISP conflicts:", err);
      }
    }
  }

  return {
    Metadata: metadata,
    VpnType: settings.vpnType,
    [vpnParamsPropName]: vpnParamsObj,
    ManualDNS: manualDNS,
    FirewallOn: fwOn,

    // Can use IPv6 connection inside tunnel
    // IPv6 has higher priority, if it supported by a server - we will use IPv6.
    // If IPv6 does not supported by server - we will use IPv4
    IPv6: settings.enableIPv6InTunnel,
    // Use ONLY IPv6 hosts (use IPv6 connection inside tunnel)
    // (ignored when IPv6!=true)
    IPv6Only: settings.showGatewaysWithoutIPv6 != true,
  };
}
