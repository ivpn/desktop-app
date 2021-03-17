import { Platform, PlatformEnum } from "@/platform/platform";

import store from "@/store";

import path from "path";

const fs = require("fs");
const os = require("os");

export function ReadAndDeleteOldSettingsIfExists() {
  // NOTE: not importing parameters
  // Selected server (together with FastestServer)
  // Connection port
  // OpenVPN proxy parameters

  const oldSettings = readOldSettings();
  if (!oldSettings) return null;

  //console.log(oldSettings);

  let origSettings = store.state.settings;
  let settingsToMerge = {};

  // copy value from `oldSettings` to `settingsToMerge` (only if destProp is exist in `origSettings`)
  let copyProp = function(srcPropName, destPropName, isBool) {
    if (
      oldSettings[srcPropName] === undefined ||
      origSettings[destPropName] === undefined
    ) {
      console.debug("Parameter not exist:", srcPropName, "->", destPropName);
      return false;
    }
    if (!isBool) settingsToMerge[destPropName] = oldSettings[srcPropName];
    else {
      if (oldSettings[srcPropName] == 0) settingsToMerge[destPropName] = false;
      else settingsToMerge[destPropName] = true;
    }
  };

  const boolParam = true;
  copyProp("VpnProtocolType", "vpnType");
  copyProp("IsMultiHop", "isMultiHop", boolParam);
  copyProp("AutoConnectOnStart", "autoConnectOnLaunch", boolParam);
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

  let newWifiConf = {};
  if (oldSettings["ServiceConnectOnInsecureWifi"] !== undefined) {
    newWifiConf.connectVPNOnInsecureNetwork =
      oldSettings["ServiceConnectOnInsecureWifi"] == 0 ? false : true;
  }

  // trusted networks config
  try {
    if (oldSettings.NetworkActions) {
      let oldWifiConf = JSON.parse(oldSettings.NetworkActions);

      newWifiConf.trustedNetworksControl =
        oldSettings.IsNetworkActionsEnabled == 1;
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

  return settingsToMerge;
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
