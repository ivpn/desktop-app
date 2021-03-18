import { Platform, PlatformEnum } from "@/platform/platform";
import { VpnTypeEnum } from "@/store/types";
import store from "@/store";

import path from "path";

const fs = require("fs");
const os = require("os");

export function ImportAndDeleteOldSettingsIfExists(mergeMethod) {
  if (!mergeMethod) return;
  // NOTE: not importing parameters
  // Fastest server settings
  // Connection port
  // OpenVPN proxy parameters

  const old = readOldSettings();
  if (!old) return null;

  let origSettings = store.state.settings;
  let settingsToMerge = {};

  // copy value from `oldSettings` to `settingsToMerge` (only if destProp is exist in `origSettings`)
  let copyProp = function(srcPropName, destPropName, isBool) {
    if (
      old[srcPropName] === undefined ||
      origSettings[destPropName] === undefined
    ) {
      console.debug("Parameter not exist:", srcPropName, "->", destPropName);
      return false;
    }
    if (!isBool) settingsToMerge[destPropName] = old[srcPropName];
    else {
      if (old[srcPropName] == 0) settingsToMerge[destPropName] = false;
      else settingsToMerge[destPropName] = true;
    }
  };

  const boolParam = true;
  copyProp("VpnProtocolType", "vpnType");
  copyProp("IsMultiHop", "isMultiHop", boolParam);
  copyProp("IsLoggingEnabled", "logging", boolParam);
  copyProp("ServiceUseObfsProxy", "connectionUseObfsproxy", boolParam);
  copyProp("FirewallAutoOnOff", "firewallActivateOnConnect", boolParam);
  copyProp("FirewallAutoOnOff", "firewallDeactivateOnDisconnect", boolParam);
  copyProp("IsAntiTracker", "isAntitracker", boolParam);
  copyProp("IsAntiTrackerHardcore", "isAntitrackerHardcore", boolParam);
  copyProp("IsCustomDns", "dnsIsCustom", boolParam);
  copyProp("CustomDns", "dnsCustom");
  copyProp("MacIsShowIconInSystemDock", "showAppInSystemDock", boolParam);
  copyProp("DoNotShowDialogOnAppClose", "quitWithoutConfirmation", boolParam);
  copyProp("DoNotShowDialogOnAppClose", "disconnectOnQuit", boolParam);

  let _svrId = null;
  let _svrExitId = null;
  const wgType = VpnTypeEnum.WireGuard;
  // macos and windows use different name for some parameters
  if (Platform() === PlatformEnum.macOS) {
    copyProp("AutoConnectOnStart", "autoConnectOnLaunch", boolParam);

    if (settingsToMerge.vpnType === wgType) {
      if (old.LastUsedWgServerId) _svrId = old.LastUsedWgServerId;
    } else {
      if (old.LastUsedServerId) _svrId = old.LastUsedServerId;
      if (old.LastUsedExitServerId) _svrExitId = old.LastUsedExitServerId;
    }
  } else if (Platform() === PlatformEnum.Windows) {
    copyProp("AutoConnectOnLaunch", "autoConnectOnLaunch", boolParam);

    if (settingsToMerge.vpnType === wgType) {
      if (old.LastUsedWgServerId) _svrId = old.LastUsedWgServerId;
    } else {
      if (old.ServerId) _svrId = old.ServerId;
      if (old.ExitServerId) _svrExitId = old.ExitServerId;
    }
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
  if (_svrId || _svrExitId) {
    let unsubscribeMethod = store.subscribe(mutation => {
      try {
        if (mutation.type === "vpnState/servers" && (_svrId || _svrExitId)) {
          const hashedSvrs = store.state.vpnState.serversHashed;
          if (_svrId) {
            let svr = hashedSvrs[_svrId];
            if (svr) store.commit("settings/serverEntry", svr);

            let svrEx = hashedSvrs[_svrExitId];
            if (svrEx) store.commit("settings/serverExit", svrEx);
          }

          _svrId = null;
          _svrExitId = null;
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
    fs.unlinkSync(settingsOldFile);

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
    fs.unlinkSync(settingsOldFile);

    try {
      fs.unlinkSync(fPathOldVer);
    } catch {
      // ignore exceptions
    }

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
