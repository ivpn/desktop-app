import { Platform, PlatformEnum } from "@/platform/platform";
import { VpnTypeEnum, Ports } from "@/store/types";
import store from "@/store";

import path from "path";

const fs = require("fs");
const os = require("os");

export function ImportAndDeleteOldSettingsIfExists(mergeMethod) {
  if (!mergeMethod) return;
  // NOTE: not importing parameters
  // OpenVPN proxyPassword (and 'Auto' proxy type)

  const old = readOldSettings();
  if (!old) return null;

  let origSettings = store.state.settings;
  let settingsToMerge = {};

  const pTypeBool = 1;
  const pTypeBoolInvert = 2;
  const pTypeString = 3;

  // copy value from `oldSettings` to `settingsToMerge` (only if destProp is exist in `origSettings`)
  let copyProp = function(srcPropName, destPropName, pType) {
    if (
      old[srcPropName] === undefined ||
      origSettings[destPropName] === undefined
    ) {
      console.debug("Parameter not exist:", srcPropName, "->", destPropName);
      return false;
    }

    switch (pType) {
      case pTypeBool:
        if (old[srcPropName] == 0) settingsToMerge[destPropName] = false;
        else settingsToMerge[destPropName] = true;
        break;
      case pTypeBoolInvert:
        if (old[srcPropName] == 0) settingsToMerge[destPropName] = true;
        else settingsToMerge[destPropName] = false;
        break;
      case pTypeString:
        if (typeof myVar === "string")
          settingsToMerge[destPropName] = old[srcPropName];
        else settingsToMerge[destPropName] = `${old[srcPropName]}`;
        break;
      default:
        settingsToMerge[destPropName] = old[srcPropName];
        break;
    }
  };

  copyProp("VpnProtocolType", "vpnType");
  copyProp("IsMultiHop", "isMultiHop", pTypeBool);
  copyProp("IsLoggingEnabled", "logging", pTypeBool);
  copyProp("ServiceUseObfsProxy", "connectionUseObfsproxy", pTypeBool);
  copyProp("FirewallAutoOnOff", "firewallActivateOnConnect", pTypeBool);
  copyProp("FirewallAutoOnOff", "firewallDeactivateOnDisconnect", pTypeBool); //FirewallDisableAutoOnOff
  copyProp("IsAntiTracker", "isAntitracker", pTypeBool);
  copyProp("IsAntiTrackerHardcore", "isAntitrackerHardcore", pTypeBool);
  copyProp("IsCustomDns", "dnsIsCustom", pTypeBool);
  copyProp("CustomDns", "dnsCustom", pTypeString);
  copyProp("MacIsShowIconInSystemDock", "showAppInSystemDock", pTypeBool);
  copyProp("DoNotShowDialogOnAppClose", "quitWithoutConfirmation", pTypeBool);
  copyProp("DoNotShowDialogOnAppClose", "disconnectOnQuit", pTypeBool);

  copyProp("ProxyPort", "ovpnProxyPort");
  copyProp("ProxyUsername", "ovpnProxyUser", pTypeString); // (ProxySafePassword (mac) + ProxyPassword (Windows) can not be imported because it is encrypted)

  let _svrId = null;
  let _svrExitId = null;
  const wgType = VpnTypeEnum.WireGuard;
  // macos and windows use different name for some parameters
  if (Platform() === PlatformEnum.macOS) {
    copyProp("FirewallAutoOnOff", "firewallActivateOnConnect", pTypeBool);
    copyProp("FirewallAutoOnOff", "firewallDeactivateOnDisconnect", pTypeBool);

    copyProp("ProxyServer", "ovpnProxyServer", pTypeString);
    copyProp("AutoConnectOnStart", "autoConnectOnLaunch", pTypeBool);

    // ovpnProxyType
    switch (old.ProxyType) {
      case 0: // None
        break;
      case 1: // Auto
        break;
      case 2: // Http
        settingsToMerge.ovpnProxyType = "http";
        break;
      case 3: // Socks
        settingsToMerge.ovpnProxyType = "socks";
        break;
      default:
        break;
    }

    // selected servers
    if (settingsToMerge.vpnType === wgType) {
      if (old.LastUsedWgServerId) _svrId = old.LastUsedWgServerId;
    } else {
      if (old.LastUsedServerId) _svrId = old.LastUsedServerId;
      if (old.LastUsedExitServerId) _svrExitId = old.LastUsedExitServerId;
    }
  } else if (Platform() === PlatformEnum.Windows) {
    copyProp(
      "FirewallDisableAutoOnOff",
      "firewallActivateOnConnect",
      pTypeBoolInvert
    );
    copyProp(
      "FirewallDisableAutoOnOff",
      "firewallDeactivateOnDisconnect",
      pTypeBoolInvert
    );

    copyProp("ProxyAddress", "ovpnProxyServer", pTypeString);
    copyProp("AutoConnectOnLaunch", "autoConnectOnLaunch", pTypeBool);

    // ovpnProxyType
    if (old.ProxyType === "http" || old.ProxyType === "socks")
      settingsToMerge.ovpnProxyType = old.ProxyType;

    // selected servers
    if (settingsToMerge.vpnType === wgType) {
      if (old.LastUsedWgServerId) _svrId = old.LastUsedWgServerId;
    } else {
      if (old.ServerId) _svrId = old.ServerId;
      if (old.ExitServerId) _svrExitId = old.ExitServerId;
    }
  }

  // selected port
  try {
    if (old.PreferredPortIndex && old.WireGuardPreferredPortIndex) {
      const p = {
        OpenVPN: Ports.OpenVPN[old.PreferredPortIndex],
        WireGuard: Ports.WireGuard[old.WireGuardPreferredPortIndex]
      };
      settingsToMerge.port = p;
    }
  } catch (e) {
    console.warn("Error importing port settings from old installation:", e);
  }

  // detect FastestServer selection
  let _fastestSvrId = null;
  if (settingsToMerge.vpnType === wgType) {
    if (old.LastWgFastestServerId) _fastestSvrId = old.LastWgFastestServerId;
  } else {
    if (old.LastOvpnFastestServerId)
      _fastestSvrId = old.LastOvpnFastestServerId;
  }
  settingsToMerge.isFastestServer = _svrId && _svrId === _fastestSvrId;

  let newWifiConf = {};
  if (old["ServiceConnectOnInsecureWifi"] !== undefined) {
    newWifiConf.connectVPNOnInsecureNetwork =
      old["ServiceConnectOnInsecureWifi"] == 0 ? false : true;
  }

  let _fastestSvrList = {};
  let _fastestSvrListSkipOvpn = true;
  let _fastestSvrListSkipWg = true;
  // fastest server list configuration
  try {
    if (old.ServersFilter) {
      const sf = JSON.parse(old.ServersFilter);
      const svrs = sf.__FastestServersInUse;
      if (svrs) {
        if (svrs.WireGuard && svrs.WireGuard.length > 0) {
          _fastestSvrListSkipWg = false;
          svrs.WireGuard.forEach(element => {
            _fastestSvrList[element] = VpnTypeEnum.WireGuard;
          });
        }
        if (svrs.OpenVPN && svrs.OpenVPN.length > 0) {
          svrs.OpenVPN.forEach(element => {
            _fastestSvrListSkipOvpn = false;
            _fastestSvrList[element] = VpnTypeEnum.OpenVPN;
          });
        }
      }
    }
  } catch (e) {
    console.warn(
      "Error importing FastestServer settings from old installation:",
      e
    );
  }

  // trusted networks config
  try {
    if (old.NetworkActions) {
      let oldWifiConf = JSON.parse(old.NetworkActions);

      newWifiConf.trustedNetworksControl = old.IsNetworkActionsEnabled == 1;
      newWifiConf.actions = {
        unTrustedConnectVpn: oldWifiConf.UnTrustedConnectToVPN,
        unTrustedEnableFirewall: oldWifiConf.UnTrustedEnableKillSwitch,

        trustedDisconnectVpn: oldWifiConf.TrustedDisconnectFromVPN,
        trustedDisableFirewall: oldWifiConf.TrustedDisableKillSwitch
      };

      // defaultTrustStatusTrusted
      newWifiConf.defaultTrustStatusTrusted = null;
      if (oldWifiConf.DefaultActionType == 2)
        newWifiConf.defaultTrustStatusTrusted = false;
      else if (oldWifiConf.DefaultActionType == 3)
        newWifiConf.defaultTrustStatusTrusted = true;

      // configured networks list
      if (oldWifiConf.Actions) {
        let newNetworks = [];
        oldWifiConf.Actions.forEach(action => {
          if (!action.Network || !action.Network.SSID || !action.Action) return;

          let isTrusted = false;
          if (action.Action == 2) isTrusted = false;
          else if (action.Action == 3) isTrusted = true;
          else return;

          let newNetwork = {
            ssid: action.Network.SSID,
            isTrusted: isTrusted
          };
          newNetworks.push(newNetwork);
        });
        if (newNetworks.length > 0) newWifiConf.networks = newNetworks;
      }
    }

    if (Object.keys(newWifiConf).length > 0) settingsToMerge.wifi = newWifiConf;
  } catch (e) {
    console.warn("Error importing WIFI settings from old installation:", e);
  }

  // integrating imported data into settings
  try {
    // integrate old settings into ours
    if (settingsToMerge && Object.keys(settingsToMerge).length > 0) {
      mergeMethod(settingsToMerge);
    }
    console.log("Old configuration import DONE.");
  } catch (e) {
    _svrId = null;
    _svrExitId = null;
    console.log("ERROR importing settings from old installation:", e);
  }

  // selected servers can not be imported immediately (because servers list still not initialized)
  // we keeping this properties locally until "vpnState/servers" will be updated
  if (_svrId || _svrExitId || (_fastestSvrList && _fastestSvrList.length > 0)) {
    let unsubscribeMethod = store.subscribe(mutation => {
      try {
        if (mutation.type === "vpnState/servers" && (_svrId || _svrExitId)) {
          const hashedSvrs = store.state.vpnState.serversHashed;
          // entry/exit servers
          if (_svrId) {
            let svr = hashedSvrs[_svrId];
            if (svr) store.commit("settings/serverEntry", svr);

            let svrEx = hashedSvrs[_svrExitId];
            if (svrEx) store.commit("settings/serverExit", svrEx);
          }

          // fastest server list config
          if (_fastestSvrList && Object.keys(_fastestSvrList).length > 0) {
            let __excludeSvrs = [];

            for (var prop in hashedSvrs) {
              const isOvpnSvr = hashedSvrs[prop].hosts === undefined;
              if (
                (isOvpnSvr && _fastestSvrListSkipOvpn) ||
                (!isOvpnSvr && _fastestSvrListSkipWg)
              )
                continue;

              if (_fastestSvrList[prop] === undefined) {
                __excludeSvrs.push(prop);
              }
            }
            if (__excludeSvrs.length > 0)
              store.commit("settings/serversFastestExcludeList", __excludeSvrs);
          }

          _svrId = null;
          _svrExitId = null;
          _fastestSvrList = null;
          // we do not need subscription anymore
          if (unsubscribeMethod) unsubscribeMethod();
        }
      } catch (e) {
        console.error(
          `Error in ImportAndDeleteOldSettingsIfExists (store.subscribe):`,
          e
        );
      }
    });
  }
}

function readOldSettings() {
  if (Platform() === PlatformEnum.Windows) return readOldSettingsWindows();
  if (Platform() === PlatformEnum.macOS) return readOldSettingsMacOS();
  return null;
}

function readOldSettingsMacOS() {
  const settingsOldFile = path.join(
    os.homedir(),
    "/Library/Preferences/net.ivpn.client.IVPN.plist"
  );

  if (!fs.existsSync(settingsOldFile)) return null;

  var execSync = require("child_process").execSync;
  try {
    let output = execSync("defaults read net.ivpn.client.IVPN").toString();
    output = output.replaceAll("\\\\", "\\");

    // remove old settings (to not import it next time)
    // fs.unlinkSync(settingsOldFile);

    const arrayOfLines = output.match(/[^\r\n]+/g);
    var re = new RegExp("([a-zA-Z0-9]+) = (.*);");

    let retObj = {};
    arrayOfLines.forEach(element => {
      const match = re.exec(element);
      if (match && match[1] && match[2]) {
        try {
          retObj[match[1]] = JSON.parse(match[2]);
        } catch (e) {
          console.warn(`Old settings parameter "${element}" parsing error:`, e);
          retObj[match[1]] = match[2];
        }
      }
    });

    return retObj;
  } catch (e) {
    console.warn("Failed to import old settings: ", e);
  }
  return null;
}

function readOldSettingsWindows() {
  try {
    const fPathOldVer = "C:\\Program Files\\IVPN Client\\old.ver";

    if (!fs.existsSync(fPathOldVer)) return null;
    const oldVer = fs.readFileSync(fPathOldVer, "utf8");

    // C:\Users\<USER>\AppData\Local\IVPN_Limited\IVPN_Client.exe_Url_2dhygxwi22dge5p2fgmqhjirdotrmd3i\<VERSION>\user.config
    const settingsOldFile = path.join(
      process.env.APPDATA,
      "..", // APPDATA returns 'C:\Users\<USER>\AppData\Roaming' but we need 'C:\Users\<USER>\AppData\Local'
      "Local",
      "IVPN_Limited",
      "IVPN_Client.exe_Url_2dhygxwi22dge5p2fgmqhjirdotrmd3i",
      oldVer,
      "user.config"
    );

    if (!fs.existsSync(settingsOldFile)) return null;
    const data = fs.readFileSync(settingsOldFile, "utf8");

    // remove old settings (to not import it next time)
    // fs.unlinkSync(settingsOldFile);

    var options = {
      ignoreAttributes: false,
      parseAttributeValue: false,
      parseNodeValue: true,
      trimValues: true
    };

    var parser = require("fast-xml-parser");
    var jsonObj = parser.parse(data, options);

    const settings =
      jsonObj.configuration.userSettings["IVPN.Properties.Settings"].setting;

    let retObj = {};
    settings.forEach(element => {
      try {
        if (element.value === "False") retObj[element["@_name"]] = 0;
        else if (element.value === "True") retObj[element["@_name"]] = 1;
        else retObj[element["@_name"]] = element.value;
      } catch (e) {
        console.warn(`Old settings parameter "${element}" reding error:`, e);
      }
    });

    return retObj;
  } catch (e) {
    console.warn("Failed to import old settings: ", e);
  }
  return null;
}
