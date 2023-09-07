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

const log = require("electron-log");
const fs = require("fs");
const net = require("net");

import { Platform, PlatformEnum } from "@/platform/platform";
import { API_SUCCESS } from "@/api/statuscode";
import { IsNewVersion } from "@/app-updater/helper";
import config from "@/config";

import { GetPortInfoFilePath } from "@/helpers/main_platform";
import { InitConnectionParamsObject } from "@/daemon-client/connectionParams.js";

import {
  VpnTypeEnum,
  VpnStateEnum,
  DaemonConnectionType,
  DnsEncryption,
  PortTypeEnum,
} from "@/store/types";
import store from "@/store";

const PingServersTimeoutMs = 4000;

const DefaultResponseTimeoutMs = 3 * 60 * 1000;

// Socket to connect to a daemon
let socket = new net.Socket();
// Request number (increasing each new request)
let requestNo = 0;
// Array response waiters
const waiters = [];

let ParanoidModeSecret = "";

const daemonRequests = Object.freeze({
  EmptyReq: "EmptyReq",

  Hello: "Hello",
  APIRequest: "APIRequest",

  GenerateDiagnostics: "GenerateDiagnostics",

  PingServers: "PingServers",
  GetServers: "GetServers",
  CheckAccessiblePorts: "CheckAccessiblePorts",
  SessionNew: "SessionNew",
  SessionDelete: "SessionDelete",
  AccountStatus: "AccountStatus",

  WiFiSettings: "WiFiSettings",
  ConnectSettings: "ConnectSettings",
  Connect: "Connect",
  Disconnect: "Disconnect",
  PauseConnection: "PauseConnection",
  ResumeConnection: "ResumeConnection",
  GetVPNState: "GetVPNState",

  KillSwitchGetStatus: "KillSwitchGetStatus",
  KillSwitchSetEnabled: "KillSwitchSetEnabled",
  KillSwitchSetAllowApiServers: "KillSwitchSetAllowApiServers",
  KillSwitchSetAllowLANMulticast: "KillSwitchSetAllowLANMulticast",
  KillSwitchSetAllowLAN: "KillSwitchSetAllowLAN",
  KillSwitchSetIsPersistent: "KillSwitchSetIsPersistent",
  KillSwitchSetUserExceptions: "KillSwitchSetUserExceptions",

  SplitTunnelGetStatus: "SplitTunnelGetStatus",
  SplitTunnelSetConfig: "SplitTunnelSetConfig",
  SplitTunnelAddApp: "SplitTunnelAddApp",
  SplitTunnelRemoveApp: "SplitTunnelRemoveApp",
  SplitTunnelAddedPidInfo: "SplitTunnelAddedPidInfo",
  GetInstalledApps: "GetInstalledApps",
  GetAppIcon: "GetAppIcon",

  SetAlternateDns: "SetAlternateDns",
  GetDnsPredefinedConfigs: "GetDnsPredefinedConfigs",

  WireGuardGenerateNewKeys: "WireGuardGenerateNewKeys",
  WireGuardSetKeysRotationInterval: "WireGuardSetKeysRotationInterval",

  SetPreference: "SetPreference",
  SetUserPreferences: "SetUserPreferences",

  WiFiAvailableNetworks: "WiFiAvailableNetworks",
  WiFiCurrentNetwork: "WiFiCurrentNetwork",

  ParanoidModeSetPasswordReq: "ParanoidModeSetPasswordReq",
});

const daemonResponses = Object.freeze({
  ErrorResp: "ErrorResp",
  ErrorRespDelayed: "ErrorRespDelayed",

  EmptyResp: "EmptyResp",
  HelloResp: "HelloResp",
  APIResponse: "APIResponse",

  SettingsResp: "SettingsResp",
  DiagnosticsGeneratedResp: "DiagnosticsGeneratedResp",

  VpnStateResp: "VpnStateResp",
  ConnectedResp: "ConnectedResp",
  DisconnectedResp: "DisconnectedResp",
  ServerListResp: "ServerListResp",
  PingServersResp: "PingServersResp",
  CheckAccessiblePortsResponse: "CheckAccessiblePortsResponse",
  SetAlternateDNSResp: "SetAlternateDNSResp",
  DnsPredefinedConfigsResp: "DnsPredefinedConfigsResp",
  KillSwitchStatusResp: "KillSwitchStatusResp",
  AccountStatusResp: "AccountStatusResp",

  SplitTunnelStatus: "SplitTunnelStatus",
  SplitTunnelAddAppCmdResp: "SplitTunnelAddAppCmdResp",
  InstalledAppsResp: "InstalledAppsResp",
  AppIconResp: "AppIconResp",

  WiFiAvailableNetworksResp: "WiFiAvailableNetworksResp",
  WiFiCurrentNetworkResp: "WiFiCurrentNetworkResp",

  ServiceExitingResp: "ServiceExitingResp",
});

export const AppUpdateInfoType = Object.freeze({
  Default: "default",
  Manual: "manual",
  Beta: "beta",
});

const ErrorRespTypes = Object.freeze({
  ErrorUnknown: 0,
  ErrorParanoidModePasswordError: 1,
});

var messageBoxFunction = null;
function RegisterMsgBoxFunc(mbFunc) {
  messageBoxFunction = mbFunc;
}
async function messageBox(cfg) {
  if (!cfg || !messageBoxFunction) return null;
  return await messageBoxFunction(cfg);
}

// JavaScript does not support int64 (and do not know how to serialize it)
// Here we are serializing BigInt manually (if necessary)
function toJson(data) {
  if (data === undefined) {
    return new Error("Nothing to serialize (object undefined)");
  }

  let intCount = 0;
  let repCount = 0;

  const json = JSON.stringify(data, (_, v) => {
    if (typeof v === "bigint") {
      intCount += 1;
      return `${v}#bigint`;
    }
    return v;
  });

  const res = json.replace(/"(-?\d+)#bigint"/g, (_, a) => {
    repCount += 1;
    return a;
  });

  if (repCount > intCount) {
    // You have a string somewhere that looks like "123#bigint";
    throw new Error(
      "BigInt serialization pattern conflict with a string value."
    );
  }

  return res;
}

// send request to connected daemon
function send(request, reqNo) {
  if (socket == null) throw Error("Unable to send request (socket is closed)");

  if (!request.ProtocolSecret) {
    request.ProtocolSecret = ParanoidModeSecret;
  }

  if (!request.Command) {
    throw Error('Unable to send request ("Command" parameter not defined)');
  }
  if (typeof request.Command === "undefined") {
    throw Error(
      'Unable to send request. Unknown command: "' + request.Command + '"'
    );
  }

  if (typeof reqNo === "undefined") {
    requestNo += 1;
    reqNo = requestNo;
  }
  request.Idx = reqNo;

  let serialized = toJson(request);
  // : Full logging is only for debug. Must be removed from production!
  //log.debug(`==> ${serialized}`);
  if (request.Command == "APIRequest")
    log.debug(`==> ${request.Command}  [${request.Idx}] ${request.APIPath}`);
  else log.debug(`==> ${request.Command}  [${request.Idx}]`);
  socket.write(`${serialized}\n`);

  return request.Idx;
}

function addWaiter(waiter, timeoutMs) {
  // create signaling promise
  const promise = new Promise((resolve, reject) => {
    // 'resolve' will be called in 'processResponse()'
    waiter.promiseResolve = resolve;
    waiter.promiseReject = reject;

    // remove waiter after timeout
    setTimeout(
      () => {
        for (let i = 0; i < waiters.length; i += 1) {
          if (waiters[i] === waiter) {
            waiters.splice(i, 1);
            reject(
              new Error("Response timeout (no response from the daemon).")
            );
            break;
          }
        }
      },

      timeoutMs != null && timeoutMs > 0 ? timeoutMs : DefaultResponseTimeoutMs
    );
  });

  // register new waiter
  waiters.push(waiter);

  return promise;
}

// If 'waitRespCommandsList' defined - the waiter will accept ANY response
// which mach one of elements in 'waitRespCommandsList'.
// Otherwise, waiter will accept only response with correspond response index.
function sendRecv(request, waitRespCommandsList, timeoutMs) {
  requestNo += 1;

  const waiter = {
    responseNo: requestNo,
    waitForCommandsList: waitRespCommandsList,
  };

  if (socket == null) {
    return new Promise((resolve, reject) => {
      reject(new Error("Error: Daemon is not connected"));
    });
  }

  let promise = addWaiter(waiter, timeoutMs);

  // send data
  try {
    send(request, requestNo);
  } catch (e) {
    console.error(e);
    throw e;
  }

  return promise;
}

function commitSession(sessionRespObj) {
  if (sessionRespObj == null) return;
  const session = {
    AccountID: sessionRespObj.AccountID,
    Session: sessionRespObj.Session,
    WgPublicKey: sessionRespObj.WgPublicKey,
    WgLocalIP: sessionRespObj.WgLocalIP,
    WgUsePresharedKey: sessionRespObj.WgUsePresharedKey,
    WgKeyGenerated: new Date(sessionRespObj.WgKeyGenerated * 1000),
    WgKeysRegenIntervalSec: sessionRespObj.WgKeysRegenInerval, // note! spelling error in received parameter name
  };
  store.commit(`account/session`, session);
  if (session.Session)
    store.commit("settings/isExpectedAccountToBeLoggedIn", true);
  return session;
}

function requestGeoLookupAsync() {
  setTimeout(async () => {
    try {
      await GeoLookup();
    } catch (e) {
      console.log(e);
    }
  }, 0);
}

function doResetSettings() {
  // Necessary to initialize selected VPN servers
  store.dispatch("settings/resetToDefaults");
}

async function processResponse(response) {
  const obj = JSON.parse(response);

  if (obj != null && obj.Command != null) {
    // TODO: Full logging is only for debug. Must be removed from production!
    //log.log(`<== ${obj.Command} ${response.length > 512 ? " ..." : response}`);
    //log.log(`<== ${response}`);
    if (obj.Command == "APIResponse")
      log.debug(
        `<== ${obj.Command}  [${obj.Idx}] ${obj.APIPath}` +
          (obj.Error ? " Error!" : "")
      );
    else log.debug(`<== ${obj.Command} [${obj.Idx}]`);
  } else log.error(`<== ${response}`);

  if (obj == null || obj.Command == null || obj.Command.length <= 0) return;

  switch (obj.Command) {
    case daemonResponses.HelloResp:
      store.commit("daemonVersion", obj.Version);
      store.commit("daemonProcessorArch", obj.ProcessorArch);

      if (obj.SettingsSessionUUID) {
        const ssID = obj.SettingsSessionUUID;
        // SettingsSessionUUID allows to detect situations when settings was erased
        // This value should be the same as on daemon side. If it differs - current settings should be erased to default state
        if (ssID != store.state.settings.SettingsSessionUUID) {
          // UUID not equal to the UUID received from daemon: reset settings
          if (store.state.settings.SettingsSessionUUID) doResetSettings();
          // save new UUID received from daemon
          store.commit("settings/settingsSessionUUID", ssID);
        }
      }

      // Check minimal required daemon version
      if (IsNewVersion(obj.Version, config.MinRequiredDaemonVer)) {
        store.commit("daemonIsOldVersionError", true);
        return;
      }
      store.commit("daemonIsOldVersionError", false);

      commitSession(obj.Session);

      // request account status update every app start
      if (store.getters["account/isLoggedIn"]) AccountStatus();

      if (obj.DisabledFunctions) {
        store.commit("disabledFunctions", obj.DisabledFunctions);
        if (obj.DisabledFunctions.WireGuardError) {
          // not able to use WG. Set OpenVPN as a default protocol
          store.commit("settings/vpnType", VpnTypeEnum.OpenVPN);
        }
      }

      if (obj.DaemonSettings) {
        store.dispatch("settings/daemonSettings", obj.DaemonSettings);

        if (obj.DaemonSettings.AntiTracker)
          store.dispatch(
            "settings/antiTracker",
            obj.DaemonSettings.AntiTracker
          );
      }

      {
        // Save DNS abilities
        store.commit("dnsAbilities", obj.Dns);
        // If the dns abilities does not fit the current custom dns configuration - reset custom dns configuration
        let curDnsEncryption = store.state.settings.dnsCustomCfg.Encryption;
        if (
          (curDnsEncryption === DnsEncryption.DnsOverTls &&
            obj.Dns.CanUseDnsOverTls !== true) ||
          (curDnsEncryption === DnsEncryption.DnsOverHttps &&
            obj.Dns.CanUseDnsOverHttps !== true)
        ) {
          store.commit("settings/dnsIsCustom", false);
          store.commit("settings/dnsCustomCfg", {
            DnsHost: "",
            Encryption: DnsEncryption.None,
            DohTemplate: "",
          });
        }
      }

      // save ParanoidMode status
      {
        let isPmPassError = obj.ParanoidMode.IsEnabled && !ParanoidModeSecret;
        store.commit("paranoidModeStatus", obj.ParanoidMode);
        store.commit("uiState/isParanoidModePasswordView", isPmPassError);

        if (obj.ParanoidMode.IsEnabled === false) ParanoidModeSecret = "";
      }

      // check accessible ports
      {
        if (!store.getters["account/isLoggedIn"]) {
          CheckAccessiblePorts(); // request all accessible ports
        }
      }
      break;

    case daemonResponses.SettingsResp:
      store.dispatch("settings/daemonSettings", obj);
      break;

    case daemonResponses.AccountStatusResp:
      //obj.APIStatus:       apiCode,
      //obj.APIErrorMessage: apiErrMsg,
      store.dispatch(`account/accountStatus`, obj);
      break;

    case daemonResponses.VpnStateResp:
      if (obj.StateVal == null) break;
      store.commit("vpnState/connectionState", obj.StateVal);
      break;

    case daemonResponses.ConnectedResp:
      {
        let connectionInfo = Object.assign({}, obj);
        connectionInfo.ConnectedSince = new Date(obj.TimeSecFrom1970 * 1000);

        store.dispatch(`vpnState/connectionInfo`, connectionInfo);
        store.commit("uiState/isPauseResumeInProgress", false);
        requestGeoLookupAsync();
      }
      break;

    case daemonResponses.DisconnectedResp:
      store.commit(`vpnState/disconnected`, obj.ReasonDescription);
      store.commit("vpnState/connectionState", VpnStateEnum.DISCONNECTED); // to properly raise value-changed event
      store.commit("uiState/isPauseResumeInProgress", false);

      // If IsStateInfo === true - it is not an disconneection event, it is just status info "disconnected"
      // No need to disable firewall in this case
      if (obj.IsStateInfo === true) break;

      if (store.state.settings.firewallDeactivateOnDisconnect === true) {
        await EnableFirewall(false);
      }
      requestGeoLookupAsync();
      break;

    case daemonResponses.ServerListResp:
      if (obj.VpnServers == null) break;
      store.dispatch(`vpnState/servers`, obj.VpnServers);
      break;

    case daemonResponses.PingServersResp: {
      if (obj.PingResults == null) break;
      store.dispatch(`vpnState/updatePings`, obj.PingResults);
      break;
    }

    case daemonResponses.CheckAccessiblePortsResponse:
      store.dispatch(`settings/notifyAccessiblePortsInfo`, obj.Ports);
      break;

    case daemonResponses.SetAlternateDNSResp:
      if (obj.IsSuccess == null) break;
      if (obj.IsSuccess !== true) {
        if (obj.ErrorMessage) {
          await messageBox({
            type: "error",
            buttons: ["OK"],
            message: `Failed to change DNS`,
            detail: obj.ErrorMessage,
          });
        }
        break;
      }
      store.dispatch(`vpnState/dns`, obj.Dns);
      break;

    case daemonResponses.DnsPredefinedConfigsResp:
      // NOTE: currently this code is not in use
      // Please, refer to 'RequestDnsPredefinedConfigs()' for details
      if (obj.DnsConfigs)
        store.commit(`dnsPredefinedConfigurations`, obj.DnsConfigs);
      break;

    case daemonResponses.KillSwitchStatusResp:
      if (obj) {
        delete obj.Idx;
        delete obj.Command;
      }

      store.dispatch(`vpnState/firewallState`, obj);

      if (
        store.state.location == null &&
        store.state.vpnState.connectionState === VpnStateEnum.DISCONNECTED
      ) {
        // if no geolocation info available - request geolocation
        requestGeoLookupAsync();
      }
      break;
    case daemonResponses.WiFiCurrentNetworkResp:
      store.commit(`vpnState/currentWiFiInfo`, {
        SSID: obj.SSID,
        IsInsecureNetwork: obj.IsInsecureNetwork,
      });
      break;
    case daemonResponses.WiFiAvailableNetworksResp:
      store.commit(`vpnState/availableWiFiNetworks`, obj.Networks);
      break;

    case daemonResponses.SplitTunnelStatus:
      store.commit(`vpnState/splitTunnelling`, obj);
      break;

    case daemonResponses.ServiceExitingResp:
      if (_onDaemonExitingCallback) _onDaemonExitingCallback();
      break;

    case daemonResponses.ErrorRespDelayed:
      if (obj.ErrorMessage) {
        console.log("ERROR response (delayed):", obj.ErrorMessage);

        // Wait some time before showing message (let app to start)
        setTimeout(async () => {
          await messageBox({
            type: "error",
            buttons: ["OK"],
            message: `IVPN daemon notifies of an error that occurred earlier`,
            detail: obj.ErrorMessage,
          });
        }, 3000);
      }
      break;

    case daemonResponses.ErrorResp:
      if (obj.ErrorMessage) console.log("ERROR response:", obj.ErrorMessage);

      if (obj.ErrorType == ErrorRespTypes.ErrorParanoidModePasswordError) {
        store.commit("uiState/isParanoidModePasswordView", true);
      } else {
        console.log("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!");
        console.log("!!!!!!!!!!!!!!!!!!!!!! ERROR RESP !!!!!!!!!!!!!!!!!!!!");
        console.log("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!");
      }
      break;

    default:
  }

  // Process waiters
  for (let i = 0; i < waiters.length; i += 1) {
    // check response index
    if (waiters[i].responseNo === obj.Idx) {
      if (
        obj.Command === daemonResponses.ErrorResp &&
        obj.ErrorMessage != null
      ) {
        waiters[i].promiseReject(obj.ErrorMessage);
        continue;
      }

      waiters[i].promiseResolve(obj);
      // remove waiter
      waiters.splice(i, 1);
      i -= 1;
    } else {
      // check response command
      let waitingCommands = waiters[i].waitForCommandsList;
      if (waitingCommands != null && waitingCommands.length > 0) {
        for (let c = 0; c < waitingCommands.length; c++) {
          if (waitingCommands[c] === obj.Command) {
            waiters[i].promiseResolve(obj);
            // remove waiter
            waiters.splice(i, 1);
            i -= 1;
          }
        }
      }
    }
  }
}

let receivedBuffer = "";
function onDataReceived(received) {
  if (received == "") return;
  const responses = received.toString().split("\n");

  const cnt = responses.length;
  if (cnt <= 0) return;

  // Each daemon response ends by new line ('\n') symbol
  // Therefore, the last response in 'responses' array have to be empty (because of .split("\n"))
  // If last response is not empty - this response is not fully received. He have to wait for the rest data.
  for (let i = 0; i < cnt - 1; i++) {
    let resp = receivedBuffer + responses[i];
    receivedBuffer = "";

    if (resp.length > 0) {
      try {
        processResponse(resp);
      } catch (e) {
        log.error("Error processing daemon response: ", e);
      }
    }
  }

  if (responses[cnt - 1].length > 0) {
    // Each daemon response ends by new line ('\n') symbol
    // Therefore, the last response in 'responses' array have to be empty (because of .split("\n"))
    // If last response is not empty - this response is not fully received. He have to wait for the rest data.
    if (receivedBuffer != "") receivedBuffer += responses[cnt - 1];
    else receivedBuffer = responses[cnt - 1];
  }
}

//////////////////////////////////////////////////////////////////////////////////////////
/// PUBLIC METHODS
//////////////////////////////////////////////////////////////////////////////////////////

var _onDaemonExitingCallback = null;

function makeHelloRequest(isSimpleConnect) {
  let appVersion = "";
  try {
    appVersion = `${require("electron").app.getVersion()}:Electron UI`;
  } catch (e) {
    console.error(e);
  }

  let helloReq = {
    Command: daemonRequests.Hello,
    ClientType: 0, // 0 - UI client; 1 - CLI
    Version: appVersion,
  };

  if (isSimpleConnect !== true) {
    helloReq = Object.assign(helloReq, {
      GetServersList: true,
      GetStatus: true,
      GetConfigParams: true,
      GetSplitTunnelStatus: true,
      GetWiFiCurrentState: true,
    });
  }

  return helloReq;
}

// Check is required operations to convert and synchronize the old-style settings with the daemon
function isNeedToConvertAndSyncOldSettings() {
  let settings = store.state.settings;
  // settings upgrade: 3.7.0 -> 3.8.1 ('autoConnectOnLaunch' now keeps on daemon's side)
  const oldIsAutoconnectOnLaunch = settings["autoConnectOnLaunch"];
  if (oldIsAutoconnectOnLaunch !== undefined) {
    return true;
  }

  // settings upgrade: 3.9.43 -> 3.10.x ('wifi' now keeps on daemon's side)
  const oldIsWifiSettings = settings["wifi"];
  if (oldIsWifiSettings !== undefined) {
    return true;
  }
  return false;
}

// Perform required operations to convert and synchronize the old-style settings with the daemon
async function convertAndSyncOldSettings() {
  let settings = store.state.settings;

  // settings upgrade: 3.7.0 -> 3.8.1 ('autoConnectOnLaunch' n ow keeps on daemon's side)
  const oldIsAutoconnectOnLaunch = settings["autoConnectOnLaunch"];
  if (oldIsAutoconnectOnLaunch !== undefined) {
    try {
      // send current settings to daemon
      await sendRecv({
        Command: daemonRequests.SetPreference,
        Key: "autoconnect_on_launch",
        Value: `${oldIsAutoconnectOnLaunch}`,
      });

      // forget old-style value
      delete settings["autoConnectOnLaunch"];
      store.commit("settings/replaceState", settings);
    } catch (e) {
      log.error(
        "Failed to convert old style settings (autoconnect_on_launch): " + e
      );
    }
  }

  // settings upgrade: 3.9.43 -> 3.10.x ('wifi' now keeps on daemon's side)
  const oldIsWifiSettings = settings["wifi"];
  if (oldIsWifiSettings !== undefined) {
    try {
      // send current settings to daemon
      SetWiFiSettings(JSON.parse(JSON.stringify(oldIsWifiSettings)));
      // forget old-style value
      delete settings["wifi"];
      store.commit("settings/replaceState", settings);
    } catch (e) {
      log.error("Failed to convert old style settings (wifi): " + e);
    }
  }
}

async function startNotifyDaemonOnParamsChange() {
  var timerNotifyDaemonOnParamsChange = null;

  // subscribe to changes in a store
  store.subscribe((mutation) => {
    try {
      switch (mutation.type) {
        case "settings/serverEntry":
        case "settings/serverExit":
        case "settings/isFastestServer":
        case "settings/isRandomServer":
        case "settings/isRandomExitServer":
        case "settings/setPort":
        case "settings/port":
        case "settings/isMultiHop":
        case "settings/serverEntryHostId":
        case "settings/serverExitHostId":
        case "settings/vpnType":
        case "settings/firewallActivateOnConnect":
        case "settings/enableIPv6InTunnel":
        case "settings/showGatewaysWithoutIPv6":
        case "settings/antiTracker":
        case "settings/dnsIsCustom":
        case "settings/dnsCustomCfg":
        case "settings/mtu":
        case "settings/ovpnProxyType":
        case "settings/ovpnProxyServer":
        case "settings/ovpnProxyPort":
        case "settings/ovpnProxyUser":
        case "settings/ovpnProxyPass":
        case "settings/serversFastestExcludeList":
          {
            //console.debug("Notifying daemon:", mutation.type);

            // reduce notifications amount: send notifications not often than once per second
            let tId = timerNotifyDaemonOnParamsChange;
            if (tId) {
              timerNotifyDaemonOnParamsChange = null;
              clearTimeout(tId);
            }
            timerNotifyDaemonOnParamsChange = setTimeout(() => {
              NotifyDaemonConnectionSettings();
              timerNotifyDaemonOnParamsChange = null;
            }, 1000);
          }
          break;

        default:
      }
    } catch (e) {
      console.error("Error in store subscriber:", e);
    }
  });

  // Send initial data to a daemon
  NotifyDaemonConnectionSettings();
}

async function ConnectToDaemon(setConnState, onDaemonExitingCallback) {
  _onDaemonExitingCallback = onDaemonExitingCallback;

  if (socket != null) {
    socket.destroy();
    socket = null;
  }

  if (setConnState === undefined)
    setConnState = function (state) {
      store.commit("daemonConnectionState", state);
    };

  // Read information about connection parameters from a file
  let portFile = await GetPortInfoFilePath();
  let portInfo = null;
  try {
    const connData = fs.readFileSync(portFile).toString();
    const parsed = connData.split(":");
    if (parsed.length !== 2) throw new Error("Failed to parse port-info file");
    portInfo = { port: parsed[0], secret: parsed[1] };
  } catch (e) {
    log.error(
      `DAEMON CONNECTION ERROR: Unable to obtain IVPN daemon connection parameters: ${e}`
    );
    throw e;
  }

  return new Promise((resolve, reject) => {
    if (!portInfo) {
      setConnState(DaemonConnectionType.NotConnected);
      reject("IVPN daemon connection info is unknown.");
      return;
    }

    // initialize current default state
    store.commit("vpnState/connectionState", VpnStateEnum.DISCONNECTED);

    socket = new net.Socket();
    socket.setNoDelay(true);

    socket
      .on("connect", async () => {
        let isNeedConvertSettings = isNeedToConvertAndSyncOldSettings();

        // SEND HELLO
        // eslint-disable-next-line no-undef
        const secretBInt = BigInt(`0x${portInfo.secret}`);
        let helloReq = makeHelloRequest(isNeedConvertSettings);
        helloReq.Secret = secretBInt;

        try {
          const disconnectDaemonFunc = function (err) {
            if (!err) return;
            setConnState(DaemonConnectionType.NotConnected);

            if (socket) {
              socket.destroy();
              socket = null;
            }

            log.error(err);
            reject(err); // REJECT
          };

          let svrListWaiter = {
            waitForCommandsList: [daemonResponses.ServerListResp],
          };
          let promiseWaiterServers = null;

          setConnState(DaemonConnectionType.Connecting);

          if (isNeedConvertSettings !== true)
            promiseWaiterServers = addWaiter(svrListWaiter, 11000);
          await sendRecv(helloReq, null, 10000);

          // the 'store.state.daemonVersion' and 'store.state.daemonIsOldVersionError' must be already initialized
          if (store.state.daemonIsOldVersionError === true) {
            const err = Error(
              `Unsupported IVPN Daemon version: v${store.state.daemonVersion} (minimum required v${config.MinRequiredDaemonVer})`
            );
            err.unsupportedDaemonVersion = true;
            disconnectDaemonFunc(err); // REJECT
            return;
          }

          if (isNeedConvertSettings === true) {
            await convertAndSyncOldSettings();
            helloReq = makeHelloRequest();

            promiseWaiterServers = addWaiter(svrListWaiter, 11000);
            await sendRecv(helloReq, null, 10000);
          }

          // waiting for all required responses
          try {
            await promiseWaiterServers;
          } catch (e) {
            disconnectDaemonFunc(Error(`Timeout: obtaining servers list`)); // REJECT
            return;
          }

          // Saving 'connected' state to a daemon
          setConnState(DaemonConnectionType.Connected);

          // start daemon notification about connection parameters change
          startNotifyDaemonOnParamsChange();

          PingServers();

          resolve(); // RESOLVE
        } catch (e) {
          log.error(`Error receiving Hello response: ${e}`);
          reject(e); // REJECT
        }
      })
      .on("data", onDataReceived);

    socket.on("close", () => {
      // Save 'disconnected' state
      setConnState(DaemonConnectionType.NotConnected);
      log.debug("Connection closed");

      socket = null;
    });

    socket.on("error", (e) => {
      log.error(`Connection error: ${e}`);
      reject(e);

      socket = null;
    });

    log.debug("Connecting to daemon...");
    try {
      socket.connect(parseInt(portInfo.port, 10), "127.0.0.1");
    } catch (e) {
      log.error("Daemon connection error: ", e);
    }
  });
}

async function Login(accountID, force, captchaID, captcha, confirmation2FA) {
  let resp = await sendRecv({
    Command: daemonRequests.SessionNew,
    AccountID: accountID,
    ForceLogin: force,
    CaptchaID: captchaID,
    Captcha: captcha,
    Confirmation2FA: confirmation2FA,
  });

  if (resp.APIStatus === API_SUCCESS) commitSession(resp.Session);

  // Returning whole response object (even in case of error)
  // it contains details about error
  return resp;
}

async function Logout(
  needToResetSettings,
  needToDisableFirewall,
  isCanDeleteSessionLocally
) {
  const isExpectedAccountToBeLoggedIn =
    store.state.settings.isExpectedAccountToBeLoggedIn;

  try {
    store.commit("settings/isExpectedAccountToBeLoggedIn", false);
    await sendRecv({
      Command: daemonRequests.SessionDelete,
      NeedToResetSettings: needToResetSettings,
      NeedToDisableFirewall: needToDisableFirewall,
      IsCanDeleteSessionLocally: isCanDeleteSessionLocally,
    });
  } catch (e) {
    if (isExpectedAccountToBeLoggedIn)
      store.commit("settings/isExpectedAccountToBeLoggedIn", true);
    throw e;
  }

  if (needToResetSettings === true) {
    doResetSettings();
  }
}

async function AccountStatus() {
  return await sendRecv({ Command: daemonRequests.AccountStatus });
}

async function GetAppUpdateInfo(appUpdateType) {
  try {
    let apiAlias = "";
    let apiAliasSign = "";

    if (appUpdateType === AppUpdateInfoType.Manual) {
      switch (Platform()) {
        case PlatformEnum.Windows:
          apiAlias = "updateInfo_manual_Windows";
          apiAliasSign = "updateSign_manual_Windows";
          break;
        case PlatformEnum.macOS:
          apiAlias = "updateInfo_manual_macOS";
          apiAliasSign = "updateSign_manual_macOS";
          break;
        case PlatformEnum.Linux:
          apiAlias = "updateInfo_manual_Linux";
          break;
      }
    } else if (appUpdateType === AppUpdateInfoType.Beta) {
      switch (Platform()) {
        case PlatformEnum.Windows:
          apiAlias = "updateInfo_beta_Windows";
          apiAliasSign = "updateSign_beta_Windows";
          break;
        case PlatformEnum.macOS:
          apiAlias = "updateInfo_beta_macOS";
          apiAliasSign = "updateSign_beta_macOS";
          break;
        case PlatformEnum.Linux:
          apiAlias = "updateInfo_beta_Linux";
          break;
      }
    } else {
      switch (Platform()) {
        case PlatformEnum.Windows:
          apiAlias = "updateInfo_Windows";
          apiAliasSign = "updateSign_Windows";
          break;
        case PlatformEnum.macOS:
          apiAlias = "updateInfo_macOS";
          apiAliasSign = "updateSign_macOS";
          break;
        case PlatformEnum.Linux:
          // For Linux it is not required to get update signature
          // because are not perform automatic update for Linux.
          // We just notifying users about new update available.
          // Info:
          //    Linux update is based on Linux repository (standard way for linux platforms)
          //    (all binaries are signed by PGP key)
          //apiAliasSign = "updateSign_Linux";
          apiAlias = "updateInfo_Linux";
          break;
      }
    }

    if (!apiAlias && !apiAliasSign) throw new Error("Unsupported platform");

    let updateInfoResp = await sendRecv({
      Command: daemonRequests.APIRequest,
      APIPath: apiAlias,
    });

    let updateInfoSignResp = null;
    if (apiAliasSign) {
      updateInfoSignResp = await sendRecv({
        Command: daemonRequests.APIRequest,
        APIPath: apiAliasSign,
      });
    }

    let respRaw = null;
    let signRespRaw = null;
    if (updateInfoResp) respRaw = updateInfoResp.ResponseData;
    if (updateInfoSignResp) signRespRaw = updateInfoSignResp.ResponseData;

    return {
      updateInfoRespRaw: respRaw,
      updateInfoSignRespRaw: signRespRaw,
    };
  } catch (e) {
    console.error("Failed to check latest update info: ", e);
  }
  return null;
}

var _geoLookupLastRequestId = 0;
async function GeoLookup() {
  // Save unique 'requestID'.
  // If there are already any 'doGeoLookup()' in progress - they will be stopped due to new
  _geoLookupLastRequestId += 1;

  // mark 'Checking geolookup...'
  store.commit("isRequestingLocation", true);
  store.commit("isRequestingLocationIPv6", true);

  // erase all known locations
  store.commit("location", null);
  store.commit("locationIPv6", null);

  // IPv4 request...
  doGeoLookup(_geoLookupLastRequestId);
  // IPv6 request ...
  doGeoLookup(_geoLookupLastRequestId, true);
}

async function doGeoLookup(requestID, isIPv6, isRetryTry) {
  if (isIPv6 == undefined) isIPv6 = false;

  let ipVerStr = isIPv6 ? "(IPv6)" : "(IPv4)";

  // Determining the properties names (according to 'isIPv6' parameter)
  let propName_Location = isIPv6 == true ? "locationIPv6" : "location";
  let propName_IsRequestingLocation =
    isIPv6 == true ? "isRequestingLocationIPv6" : "isRequestingLocation";

  // Function returns 'true' then we received location info in disconnected state
  let isRealGeoLocationCheck = function () {
    return (
      store.state.vpnState.connectionState === VpnStateEnum.DISCONNECTED ||
      store.getters["vpnState/isPaused"]
    );
  };

  // Set correct geo-lookup IPvX view based on the data which is already exists
  // (e.g. if there is no IPv6 data but IPv4 is already exists -> switch to IPv4 view)
  let setCorrectGeoIPView = function () {
    const isIPv6View = store.state.uiState.isIPv6View;
    if (
      isIPv6View === true &&
      !store.state.locationIPv6 &&
      store.state.location
    )
      store.commit("uiState/isIPv6View", false);
    else if (
      isIPv6View === false &&
      !store.state.location &&
      store.state.locationIPv6
    )
      store.commit("uiState/isIPv6View", true);
  };

  let retLocation = null;
  let isRealGeoLocationOnStart = isRealGeoLocationCheck();

  // mark 'Checking geolookup...'
  store.commit(propName_IsRequestingLocation, true);

  // To run new location request - the location info should be empty
  // Otherwise - skip this request (since location already known)
  if (store.state["propName_Location"] != null) {
    // un-mark 'Checking geolookup...'
    store.commit(propName_IsRequestingLocation, false);
    log.info(`The ${ipVerStr} location already defined`);
    return;
  }
  if (requestID != _geoLookupLastRequestId) {
    // un-mark 'Checking geolookup...'
    store.commit(propName_IsRequestingLocation, false);
    log.info("New API 'geo-lookup' request detected. Skipping current.");
    return;
  }

  let doNotRetry = false;
  // DO REQUEST ...
  try {
    let resp = await sendRecv({
      Command: daemonRequests.APIRequest,
      APIPath: "geo-lookup",
      IPProtocolRequired: isIPv6 ? 2 : 1, // IPvAny = 0, IPv4 = 1, IPv6 = 2
    });

    if (resp.Error !== "") {
      log.warn(`API 'geo-lookup' error: ${ipVerStr} ${resp.Error}`);

      setCorrectGeoIPView();

      const errStr = resp.Error.toLowerCase();
      // check error: if no route OR no protocol support - no sense to retry new requests
      if (
        // TODO! necessary to avoid checking text description of the golang errors
        errStr.includes("no ipv6 support") ||
        errStr.includes("no route to host")
      )
        doNotRetry = true;
    } else {
      if (isRealGeoLocationOnStart != isRealGeoLocationCheck()) {
        log.warn(`Skip geo-lookup result ${ipVerStr} (conn. state changed)`);
      } else {
        // {"ip_address":"","isp":"","organization":"","country":"","country_code":"","city":"","latitude": 0.0,"longitude":0.0,"isIvpnServer":false}
        retLocation = JSON.parse(`${resp.ResponseData}`);
        if (!retLocation || !retLocation.latitude || !retLocation.longitude) {
          log.warn(`API ERROR: bad geo-lookup response`);
          retLocation = null;
        } else {
          retLocation.isRealLocation = isRealGeoLocationOnStart;
          log.info("API: 'geo-lookup' success.");
          store.commit(propName_Location, retLocation);

          setTimeout(() => {
            setCorrectGeoIPView();
          }, 2000);
        }
      }
    }
  } catch (e) {
    log.warn(`geo-lookup error ${ipVerStr}`, e.toString());
    setCorrectGeoIPView();
  } finally {
    store.commit(propName_IsRequestingLocation, false); // un-mark 'Checking geolookup...'
  }

  if (doNotRetry == false && retLocation == null && !isRetryTry) {
    for (let r = 1; r <= 3; r++) {
      // if there already new request available - skip executing current request
      if (requestID != _geoLookupLastRequestId) {
        log.info("New API 'geo-lookup' request detected. Skipping current");
        break;
      }

      log.warn(`Geo-lookup request failed ${ipVerStr}. Retrying (${r})...`);

      let promise = new Promise((resolve) => {
        store.commit(propName_IsRequestingLocation, true); // mark 'Checking geolookup...'
        setTimeout(() => {
          if (!requestID == _geoLookupLastRequestId) {
            resolve(null);
            return;
          }
          resolve(doGeoLookup(requestID, isIPv6, true));
        }, r * 1000);
      });

      retLocation = await promise;
      if (retLocation != null) break;
    }
  }
}

async function ServersUpdateRequest() {
  send({
    Command: daemonRequests.GetServers,
    RequestServersUpdate: true, //Force to update servers from the backend (locations, hosts and hosts load)
  });
}

let pingServersPromise = null;
async function PingServers() {
  const p = pingServersPromise;
  if (p) {
    console.debug("Pinging already in progress. Waiting...");
    return await p;
  }

  let ret = null;
  store.commit("vpnState/isPingingServers", true);
  try {
    pingServersPromise = sendRecv(
      {
        Command: daemonRequests.PingServers,
        TimeOutMs: PingServersTimeoutMs,
        VpnTypePrioritization: true,
        VpnTypePrioritized: store.state.settings.vpnType,
      },
      [daemonResponses.PingServersResp]
    );

    ret = await pingServersPromise;
  } finally {
    pingServersPromise = null;
    store.commit("vpnState/isPingingServers", false);
  }
  return ret;
}

async function CheckAccessiblePorts(portsToCheck) {
  // if ports to ckeck is not defined - test only ports applicable for current selected protocol
  if (!portsToCheck) {
    let portsToCheckNormalised = store.getters["vpnState/connectionPorts"];
    portsToCheck = [];
    // we have to convert port objects to required format
    for (let p of portsToCheckNormalised) {
      portsToCheck.push({
        port: p.port,
        type: p.type == PortTypeEnum.UDP ? "UDP" : "TCP",
      });
    }
  }

  await sendRecv({
    Command: daemonRequests.CheckAccessiblePorts,
    PortsToTest: portsToCheck,
  });
}

async function GetDiagnosticLogs() {
  let logs = await sendRecv({ Command: daemonRequests.GenerateDiagnostics });

  // remove internal protocol variables
  delete logs.Command;
  delete logs.Idx;

  return logs;
}

async function NotifyDaemonConnectionSettings() {
  const paramsObj = InitConnectionParamsObject();

  await sendRecv({
    Command: daemonRequests.ConnectSettings,
    Params: paramsObj,
  });
}

// The 'Connect' method increasing this value at the method beginning and then checks this value before sending request to a daemon:
//  if the value is not equal to the value at method beginning - do not send 'Connect' request to the daemon.
// (this can happen when 'Disconnect' called OR new call of 'Connect' method)
let connectionRequestId = 0;

async function Connect() {
  const connectID = ++connectionRequestId;

  let settings = store.state.settings;

  // Switching the UI to the 'connecting' view
  // Important! This put UI in un-synchronized state with the real state of the daemon!
  //            Normally we are updating VPN state after connection/disconnection.
  //            But in case of any problems - we have to request VPN status manually (to synchronize the VPN state in UI with the daemon state)
  store.commit("vpnState/connectionState", VpnStateEnum.CONNECTING);

  // In case of "fastest server" or "random server" -> update entry-/exit- servers
  try {
    const isRandomExitSvr = store.getters["settings/isRandomExitServer"];

    // Function to filter applicable multi-hop servers: exclude servers from same Country/ISP
    const filterSvrsExcludeSameCountryIsp = function (servers, svr) {
      if (!servers) return servers;
      let tmpSvrs = servers.filter(
        (s) => s.country_code !== svr.country_code && s.isp !== svr.isp
      );
      if (tmpSvrs.length == 0) return servers; // fallback (if filtered list is empty)
      return tmpSvrs;
    };

    // ENTRY SERVER
    if (store.getters["settings/isFastestServer"]) {
      const funcGetPing = store.getters["vpnState/funcGetPing"];
      // looking for fastest server
      let fastest = store.getters["vpnState/fastestServer"];
      if (fastest == null || funcGetPing(fastest) == null) {
        // request servers ping
        console.log(
          "Connect to fastest server (fastest server not defined). Pinging servers..."
        );

        try {
          // Try to ping servers
          await PingServers();
        } catch (e) {
          console.error(e);
        }

        // fastestServer returns: the server with the lowest latency
        // (looking for the active servers that have latency info)
        // If there is no latency info for any server:
        // - return the nearest server (if geolocation info is known)
        // - else: return the currently selected server (if applicable)
        // - else: return the first server in the list (as a fallback)
        fastest = store.getters["vpnState/fastestServer"];
      }
      if (fastest != null) store.dispatch("settings/serverEntry", fastest);
    } else if (store.getters["settings/isRandomServer"]) {
      // random server
      let servers = store.getters["vpnState/activeServers"];
      if (!isRandomExitSvr) {
        servers = filterSvrsExcludeSameCountryIsp(servers, settings.serverExit);
      }
      let randomIdx = Math.floor(Math.random() * Math.floor(servers.length));
      store.dispatch("settings/serverEntry", servers[randomIdx]);
    }

    // EXIT SERVER
    if (isRandomExitSvr) {
      const servers = store.getters["vpnState/activeServers"];
      const exitServers = filterSvrsExcludeSameCountryIsp(
        servers,
        settings.serverEntry
      );
      const randomIdx = Math.floor(
        Math.random() * Math.floor(exitServers.length)
      );
      store.dispatch("settings/serverExit", exitServers[randomIdx]);
    }
  } catch (e) {
    console.error("Failed to connect: ", e);
    // We have to request VPN status manually (to synchronize the VPN state in UI with the daemon state)
    await RequestVPNState();
    return;
  }

  if (connectID != connectionRequestId) {
    console.log("Connection request cancelled");
    return;
  }

  const paramsObj = InitConnectionParamsObject();

  await sendRecv({
    Command: daemonRequests.Connect,
    Params: paramsObj,
  });
}

async function RequestVPNState() {
  await sendRecv({
    Command: daemonRequests.GetVPNState,
  });
}

async function Disconnect() {
  // Just to cancel current connection request (if we are preparing for connection now)
  ++connectionRequestId;

  if (store.state.vpnState.connectionState === VpnStateEnum.CONNECTED)
    store.commit("vpnState/connectionState", VpnStateEnum.DISCONNECTING);

  await sendRecv(
    {
      Command: daemonRequests.Disconnect,
    },
    [daemonResponses.DisconnectedResp]
  );
}

async function PauseConnection(pauseSeconds) {
  if (pauseSeconds == null) return;
  const vpnState = store.state.vpnState;
  if (vpnState.connectionState !== VpnStateEnum.CONNECTED) return;

  store.commit("uiState/isPauseResumeInProgress", true); // value resets to 'false' when ConnectedResp/DisconnectedResp received
  await sendRecv({
    Command: daemonRequests.PauseConnection,
    Duration: pauseSeconds,
  });
}

async function ResumeConnection() {
  if (!store.getters["vpnState/isPaused"]) return;

  store.commit("uiState/isPauseResumeInProgress", true); // value resets to 'false' when ConnectedResp/DisconnectedResp received
  await sendRecv({
    Command: daemonRequests.ResumeConnection,
  });
}

function throwIfForbiddenToEnableFirewall() {
  if (store.getters["vpnState/isPaused"])
    throw Error("Please, resume connection first to enable Firewall");
}

async function EnableFirewall(enable) {
  if (store.state.vpnState.firewallState.IsPersistent === true) {
    console.error("Not allowed to change firewall state in Persistent mode");
    return;
  }
  if (enable === true) {
    throwIfForbiddenToEnableFirewall();
  }

  await sendRecv({
    Command: daemonRequests.KillSwitchSetEnabled,
    IsEnabled: enable,
  });
}

async function KillSwitchGetStatus() {
  await sendRecv({
    Command: daemonRequests.KillSwitchGetStatus,
  });
}
async function KillSwitchSetAllowApiServers(IsAllowApiServers) {
  await sendRecv({
    Command: daemonRequests.KillSwitchSetAllowApiServers,
    IsAllowApiServers,
  });
}

async function KillSwitchSetAllowLANMulticast(AllowLANMulticast) {
  await sendRecv({
    Command: daemonRequests.KillSwitchSetAllowLANMulticast,
    AllowLANMulticast,
  });
}
async function KillSwitchSetAllowLAN(AllowLAN) {
  await sendRecv({
    Command: daemonRequests.KillSwitchSetAllowLAN,
    AllowLAN,
  });
}
async function KillSwitchSetIsPersistent(IsPersistent) {
  if (IsPersistent === true) {
    throwIfForbiddenToEnableFirewall();
  }
  await sendRecv({
    Command: daemonRequests.KillSwitchSetIsPersistent,
    IsPersistent,
  });
}

async function KillSwitchSetUserExceptions(userExceptions) {
  await sendRecv({
    Command: daemonRequests.KillSwitchSetUserExceptions,
    UserExceptions: userExceptions,
  });
}

async function SplitTunnelGetStatus() {
  let ret = await sendRecv(
    {
      Command: daemonRequests.SplitTunnelGetStatus,
    },
    [daemonResponses.SplitTunnelStatus]
  );
  return ret;
}
async function SplitTunnelSetConfig(IsEnabled, IsInversed, IsAnyDns, doReset) {
  await sendRecv(
    {
      Command: daemonRequests.SplitTunnelSetConfig,
      IsEnabled,
      IsInversed,
      IsAnyDns,
      Reset: doReset === true,
    },
    [daemonResponses.SplitTunnelStatus]
  );
}

async function SplitTunnelAddApp(execCmd, funcShowMessageBox) {
  let ret = await sendRecv(
    {
      Command: daemonRequests.SplitTunnelAddApp,
      Exec: execCmd,
    },
    [daemonResponses.SplitTunnelAddAppCmdResp, daemonResponses.EmptyResp]
  );

  if (ret != null && ret.Command != daemonResponses.ErrorResp) {
    // save info about added app into "apps favorite list"
    store.dispatch("settings/saveAddedAppCounter", execCmd);
  }

  // Description of Split Tunneling commands sequence to run the application:
  //	[client]					          [daemon]
  //	SplitTunnelAddApp		    ->
  //							            <-	windows:	types.EmptyResp (success)
  //							            <-	linux:		types.SplitTunnelAddAppCmdResp (some operations required on client side)
  //	<windows: done>
  // 	<execute shell command: types.SplitTunnelAddAppCmdResp.CmdToExecute and get PID>
  //  SplitTunnelAddedPidInfo	->
  // 							            <-	types.EmptyResp (success)

  if (ret != null && ret.Command == daemonResponses.EmptyResp) {
    // (Windows) Success
    return;
  }

  if (ret.Command != daemonResponses.SplitTunnelAddAppCmdResp)
    throw new Error("Unexpected response");

  if (ret != null && ret.Command == daemonResponses.SplitTunnelAddAppCmdResp) {
    if (Platform() != PlatformEnum.Linux) {
      throw new Error(
        "Launching external commands in the Split Tunnel environment is not supported for this platform"
      );
    }

    if (ret.IsAlreadyRunning && funcShowMessageBox) {
      let warningMes = ret.IsAlreadyRunningMessage;
      if (!warningMes || warningMes.length <= 0) {
        // Note! Normally, this message will be never used. The text will come from daemon in 'IsAlreadyRunningMessage'
        warningMes =
          "It appears the application is already running. Some applications must be closed before launching them in the Split Tunneling environment or they may not be excluded from the VPN tunnel.";
      }

      let msgBoxConfig = {
        type: "question",
        message: "Do you really want to launch the application?",
        detail: warningMes,
        buttons: ["Cancel", "Launch"],
      };
      let action = await funcShowMessageBox(msgBoxConfig);
      if (action.response == 0) {
        return;
      }
    }

    try {
      let XDG_CURRENT_DESKTOP = process.env["ORIGINAL_XDG_CURRENT_DESKTOP"];
      if (!XDG_CURRENT_DESKTOP)
        XDG_CURRENT_DESKTOP = process.env["XDG_CURRENT_DESKTOP"];

      //-------------------
      // For a security reasons, we are not using SplitTunnelAddAppCmdResp.CmdToExecute command
      // Instead, use hardcoded binary path to execute '/usr/bin/ivpn'
      let eaaArgs = "";
      if (ParanoidModeSecret) {
        eaaArgs = `-eaa_hash '${ParanoidModeSecret}' `;
      }
      let shellCommandToRun = `/usr/bin/ivpn exclude  ${eaaArgs}${execCmd}`;

      var exec = require("child_process").exec;
      let child = exec(shellCommandToRun, {
        env: {
          ...process.env,
          XDG_CURRENT_DESKTOP: XDG_CURRENT_DESKTOP,
          // Inform CLI that it started by the UI
          // The CLI will skip sending 'SplitTunnelAddApp' in this case
          IVPN_STARTED_BY_PARENT: "IVPN_UI",
        },
      });
      //-------------------
      //var spawn = require("child_process").spawn;
      //let child = spawn("/usr/bin/ivpn", ["exclude", execCmd], {
      //  detached: true,
      //  env: {
      //    ...process.env,
      //    XDG_CURRENT_DESKTOP: XDG_CURRENT_DESKTOP,
      //    // Inform CLI that it started by the UI
      //    // The CLI will skip sending 'SplitTunnelAddApp' in this case
      //    IVPN_STARTED_BY_PARENT: "IVPN_UI",
      //  },
      //});
      //// do not exit child process when parent application stops
      //child.unref();
      //-------------------

      console.log(
        "Started command in Split Tunnel environment: PID=",
        child.pid
      );

      // No necessary to send 'SplitTunnelAddedPidInfo'
      // It will ne sent by '/usr/bin/ivpn'
      //    await sendRecv({
      //      Command: daemonRequests.SplitTunnelAddedPidInfo,
      //      Pid: child.pid,
      //      Exec: execCmd,
      //      CmdToExecute: shellCommandToRun,
      //});
    } catch (e) {
      console.error(e);
    }
  }
}

async function SplitTunnelRemoveApp(Pid, Exec) {
  await sendRecv({
    Command: daemonRequests.SplitTunnelRemoveApp,
    Pid,
    Exec,
  });
}

async function GetInstalledApps() {
  try {
    let extraArgsJson = "";
    if (Platform() == PlatformEnum.Windows) {
      // Windows
      let appData = process.env["APPDATA"];
      if (appData)
        extraArgsJson = JSON.stringify({ WindowsEnvAppdata: appData });
    } else if (Platform() == PlatformEnum.Linux) {
      // Linux

      let XDG_CURRENT_DESKTOP = process.env["ORIGINAL_XDG_CURRENT_DESKTOP"];
      if (!XDG_CURRENT_DESKTOP)
        XDG_CURRENT_DESKTOP = process.env["XDG_CURRENT_DESKTOP"];

      // get icons theme name
      let iconsThemeName = "";
      try {
        var execSync = require("child_process").execSync;
        let envs = { ...process.env, XDG_CURRENT_DESKTOP: XDG_CURRENT_DESKTOP };
        iconsThemeName = execSync(
          "/usr/bin/gsettings get org.gnome.desktop.interface icon-theme",
          { env: envs }
        )
          .toString()
          .trim()
          .replace(/^'/, "")
          .replace(/'$/, "");
      } catch (e) {
        console.error(e);
      }

      // get environment variables

      if (process.env.IS_DEBUG) {
        XDG_CURRENT_DESKTOP = "ubuntu:GNOME";
      }

      let XDG_DATA_DIRS = process.env["XDG_DATA_DIRS"];
      let HOME = process.env["HOME"];
      extraArgsJson = JSON.stringify({
        IconsTheme: iconsThemeName,
        EnvVar_XDG_CURRENT_DESKTOP: XDG_CURRENT_DESKTOP,
        EnvVar_XDG_DATA_DIRS: XDG_DATA_DIRS,
        EnvVar_HOME: HOME,
      });
    }

    const responseTimeoutMs = 25 * 1000;
    let appsResp = await sendRecv(
      {
        Command: daemonRequests.GetInstalledApps,
        ExtraArgsJSON: extraArgsJson,
      },
      [daemonResponses.InstalledAppsResp],
      responseTimeoutMs
    );

    if (appsResp == null) {
      return null;
    }
    return appsResp.Apps;
  } catch (e) {
    console.error("GetInstalledApps failed: ", e);
    return null;
  }
}

async function GetAppIcon(binaryPath) {
  if (store.state.vpnState.splitTunnelling.IsCanGetAppIconForBinary !== true) {
    return null;
  }
  try {
    let resp = await sendRecv({
      Command: daemonRequests.GetAppIcon,
      AppBinaryPath: binaryPath,
    });

    if (resp == null) {
      return null;
    }

    return resp.AppIcon;
  } catch (e) {
    console.error("GetInstalledApps failed: ", e);
    return null;
  }
}

async function SetDNS() {
  let Dns = {
    DnsHost: "",
    Encryption: DnsEncryption.None,
    DohTemplate: "",
  };

  if (store.state.settings.dnsIsCustom) {
    Dns = store.state.settings.dnsCustomCfg;
  }

  let at = store.state.settings.antiTracker;

  // send change-request
  await sendRecv({
    Command: daemonRequests.SetAlternateDns,
    Dns,
    AntiTracker: at, // anti-tracker metadata (when enabled: has higher priority than 'Dns')
  });
}

import { GetSystemDohConfigurations } from "@/helpers/main_dns";
async function RequestDnsPredefinedConfigs() {
  //await sendRecv({
  //  Command: daemonRequests.GetDnsPredefinedConfigs,
  //});
  let retCfgs = await GetSystemDohConfigurations();
  if (retCfgs) store.commit(`dnsPredefinedConfigurations`, retCfgs);
  else store.commit(`dnsPredefinedConfigurations`, []);
}

async function SetUserPrefs(userPrefs) {
  if (!userPrefs) return;

  await sendRecv({
    Command: daemonRequests.SetUserPreferences,
    UserPrefs: userPrefs,
  });
}

async function SetAutoconnectOnLaunch(
  enable,
  isApplicableByDaemonInBackground
) {
  if (enable != null && enable != undefined) {
    await sendRecv({
      Command: daemonRequests.SetPreference,
      Key: "autoconnect_on_launch",
      Value: `${enable}`,
    });
  }

  if (
    isApplicableByDaemonInBackground != null &&
    isApplicableByDaemonInBackground != undefined
  ) {
    await sendRecv({
      Command: daemonRequests.SetPreference,
      Key: "autoconnect_on_launch_daemon",
      Value: `${isApplicableByDaemonInBackground}`,
    });
  }
}

async function SetLogging(enable) {
  const Key = "enable_logging";
  let Value = `${enable}`;

  await sendRecv({
    Command: daemonRequests.SetPreference,
    Key,
    Value,
  });
}

async function WgRegenerateKeys() {
  await sendRecv({
    Command: daemonRequests.WireGuardGenerateNewKeys,
  });
}

async function WgSetKeysRotationInterval(intervalSec) {
  await sendRecv({
    Command: daemonRequests.WireGuardSetKeysRotationInterval,
    Interval: intervalSec,
  });
}

async function SetWiFiSettings(wifiSettings) {
  await sendRecv({
    Command: daemonRequests.WiFiSettings,
    Params: wifiSettings,
  });
}

async function GetWiFiAvailableNetworks() {
  await sendRecv({
    Command: daemonRequests.WiFiAvailableNetworks,
  });
}

function paranoidModePasswordHash(password) {
  if (!password) return "";
  var pbkdf2 = require("pbkdf2");
  let hash = pbkdf2.pbkdf2Sync(password, "", 4096, 64, "sha256");
  return hash.toString("base64");
}

async function SetParanoidModePassword(newPassword, oldPassword) {
  if (!newPassword && !oldPassword) {
    // to disable PM the 'newPassword' must be empty. But we need 'oldPassword' instead
    throw "Actual password not defined";
  }

  newPassword = paranoidModePasswordHash(newPassword);
  oldPassword = paranoidModePasswordHash(oldPassword);

  ParanoidModeSecret = oldPassword;
  if (!ParanoidModeSecret) ParanoidModeSecret = newPassword;

  await sendRecv({
    Command: daemonRequests.ParanoidModeSetPasswordReq,
    NewSecret: newPassword,
    ProtocolSecret: oldPassword,
  });

  ParanoidModeSecret = newPassword;
}

async function SetLocalParanoidModePassword(password) {
  if (!password) throw "Password is empty";

  password = paranoidModePasswordHash(password);

  await sendRecv({
    Command: daemonRequests.EmptyReq,
    ProtocolSecret: password,
  });

  ParanoidModeSecret = password;
  store.commit("uiState/isParanoidModePasswordView", false);
}

export default {
  AppUpdateInfoType,

  RegisterMsgBoxFunc,
  ConnectToDaemon,

  GetDiagnosticLogs,

  Login,
  Logout,
  AccountStatus,

  GetAppUpdateInfo,

  GeoLookup,
  PingServers,
  CheckAccessiblePorts,
  ServersUpdateRequest,
  KillSwitchGetStatus,

  Connect,
  Disconnect,
  PauseConnection,
  ResumeConnection,

  EnableFirewall,
  KillSwitchSetAllowApiServers,
  KillSwitchSetAllowLANMulticast,
  KillSwitchSetAllowLAN,
  KillSwitchSetIsPersistent,
  KillSwitchSetUserExceptions,

  SplitTunnelGetStatus,
  SplitTunnelSetConfig,
  SplitTunnelAddApp,
  SplitTunnelRemoveApp,
  GetInstalledApps,
  GetAppIcon,

  SetDNS,
  RequestDnsPredefinedConfigs,

  SetUserPrefs,

  SetAutoconnectOnLaunch,
  SetLogging,
  WgRegenerateKeys,
  WgSetKeysRotationInterval,

  GetWiFiAvailableNetworks,
  SetWiFiSettings,

  SetParanoidModePassword,
  SetLocalParanoidModePassword,
};
