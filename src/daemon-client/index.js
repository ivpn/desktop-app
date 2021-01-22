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

const log = require("electron-log");
const fs = require("fs");
const net = require("net");
const os = require("os");

import { Platform, PlatformEnum } from "@/platform/platform";
import { isStrNullOrEmpty } from "@/helpers/helpers";
import { API_SUCCESS } from "@/api/statuscode";
import { IsNewerVersion } from "@/app-updater";
import config from "@/config";

import {
  VpnTypeEnum,
  VpnStateEnum,
  PauseStateEnum,
  DaemonConnectionType
} from "@/store/types";
import store from "@/store";

const PingServersTimeoutMs = 4000;
const PingServersRetriesCnt = 4;

const DefaultResponseTimeoutMs = 15 * 1000;

// Socket to connect to a daemon
let socket = new net.Socket();
// Request number (increasing each new request)
let requestNo = 0;
// Array response waiters
const waiters = [];

const daemonRequests = Object.freeze({
  Hello: "Hello",
  APIRequest: "APIRequest",

  GenerateDiagnostics: "GenerateDiagnostics",

  PingServers: "PingServers",
  SessionNew: "SessionNew",
  SessionDelete: "SessionDelete",
  AccountStatus: "AccountStatus",
  Connect: "Connect",
  Disconnect: "Disconnect",
  PauseConnection: "PauseConnection",
  ResumeConnection: "ResumeConnection",

  KillSwitchGetStatus: "KillSwitchGetStatus",
  KillSwitchSetEnabled: "KillSwitchSetEnabled",
  KillSwitchSetAllowLANMulticast: "KillSwitchSetAllowLANMulticast",
  KillSwitchSetAllowLAN: "KillSwitchSetAllowLAN",
  KillSwitchSetIsPersistent: "KillSwitchSetIsPersistent",

  SetAlternateDns: "SetAlternateDns",
  WireGuardGenerateNewKeys: "WireGuardGenerateNewKeys",
  SetPreference: "SetPreference",
  WireGuardSetKeysRotationInterval: "WireGuardSetKeysRotationInterval",

  WiFiAvailableNetworks: "WiFiAvailableNetworks",
  WiFiCurrentNetwork: "WiFiCurrentNetwork"
});

const daemonResponses = Object.freeze({
  HelloResp: "HelloResp",
  APIResponse: "APIResponse",

  ConfigParamsResp: "ConfigParamsResp",
  DiagnosticsGeneratedResp: "DiagnosticsGeneratedResp",

  VpnStateResp: "VpnStateResp",
  ConnectedResp: "ConnectedResp",
  DisconnectedResp: "DisconnectedResp",
  ServerListResp: "ServerListResp",
  PingServersResp: "PingServersResp",
  SetAlternateDNSResp: "SetAlternateDNSResp",
  KillSwitchStatusResp: "KillSwitchStatusResp",
  AccountStatusResp: "AccountStatusResp",

  WiFiAvailableNetworksResp: "WiFiAvailableNetworksResp",
  WiFiCurrentNetworkResp: "WiFiCurrentNetworkResp",

  ErrorResp: "ErrorResp"
});

// Read information about connection parameters from a file
function getDaemonConnectionParams_cb(callback) {
  let getConnectionInfoFile = function(fileCb) {
    switch (Platform()) {
      case PlatformEnum.macOS:
        fileCb(null, "/Library/Application Support/IVPN/port.txt");
        break;
      case PlatformEnum.Linux:
        fileCb(null, "/opt/ivpn/mutable/port.txt");
        break;
      case PlatformEnum.Windows: {
        let Registry = require("winreg");
        let regKey = new Registry({
          hive: Registry.HKLM,
          key: "\\Software\\IVPN Client"
        });
        regKey.get(Registry.DEFAULT_VALUE, function(err, item) {
          if (err)
            fileCb(`Error reading installation path (registry):${err}`, null);
          else fileCb(null, `${item.value}\\etc\\port.txt`);
        });
        break;
      }
      default:
        fileCb(`Not supported platform: '${os.platform()}'`, null);
    }
  };

  getConnectionInfoFile((err, path) => {
    if (err) callback(err, null);
    else {
      try {
        const connData = fs.readFileSync(path).toString();
        const parsed = connData.split(":");
        if (parsed.length !== 2)
          throw new Error("Failed to parse port-info file");
        callback(null, { port: parsed[0], secret: parsed[1] });
      } catch (e) {
        callback(
          new Error(`Unable to obtain IVPN daemon connection parameters: ${e}`),
          null
        );
      }
    }
  });
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
  if (socket == null)
    return new Error("Unable to send request (socket is closed)");

  if (typeof request.Command === "undefined") {
    return new Error(
      'Unable to send request ("Command" parameter not defined)'
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
  log.debug(`==> ${request.Command}  [${request.Idx}]`);
  socket.write(`${serialized}\n`);

  return request.Idx;
}

async function sendRecv(request, waitRespCommandsList, timeoutMs) {
  requestNo += 1;

  const waiter = {
    responseNo: requestNo,
    promiseResolve: null,
    promiseReject: null,
    waitForCommandsList: waitRespCommandsList
  };

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

  // send data
  send(request, requestNo);

  return promise;
}
function commitNoSession() {
  const session = {
    AccountID: "",
    Session: "",
    WgPublicKey: "",
    WgLocalIP: "",
    WgKeyGenerated: new Date(),
    WgKeysRegenIntervalSec: 0
  };
  commitSession(session);
}
function commitSession(sessionRespObj) {
  if (sessionRespObj == null) return;
  const session = {
    AccountID: sessionRespObj.AccountID,
    Session: sessionRespObj.Session,
    WgPublicKey: sessionRespObj.WgPublicKey,
    WgLocalIP: sessionRespObj.WgLocalIP,
    WgKeyGenerated: new Date(sessionRespObj.WgKeyGenerated * 1000),
    WgKeysRegenIntervalSec: sessionRespObj.WgKeysRegenInerval // note! spelling error in received parameter name
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

async function processResponse(response) {
  const obj = JSON.parse(response);

  if (obj != null && obj.Command != null) {
    // TODO: Full logging is only for debug. Must be removed from production!
    //log.log(`<== ${obj.Command} ${response.length > 512 ? " ..." : response}`);
    //log.log(`<== ${response}`);
    log.debug(`<== ${obj.Command} [${obj.Idx}]`);
  } else log.error(`<== ${response}`);

  if (obj == null || obj.Command == null || obj.Command.length <= 0) return;

  switch (obj.Command) {
    case daemonResponses.HelloResp:
      store.commit("daemonVersion", obj.Version);

      // Check minimal required daemon version
      if (IsNewerVersion(obj.Version, config.MinRequiredDaemonVer)) {
        store.commit("daemonIsOldVersionError", true);
        return;
      }
      store.commit("daemonIsOldVersionError", false);

      commitSession(obj.Session);

      // if no info about account status - request it
      if (
        store.getters["account/isLoggedIn"] &&
        !store.getters["account/isAccountStateExists"]
      ) {
        AccountStatus();
      }

      if (obj.DisabledFunctions != null)
        store.commit("disabledFunctions", obj.DisabledFunctions);

      break;

    case daemonResponses.ConfigParamsResp:
      store.commit("configParams", obj);
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
      store.dispatch(`vpnState/connectionInfo`, {
        VpnType: obj.VpnType,
        ConnectedSince: new Date(obj.TimeSecFrom1970 * 1000),
        ClientIP: obj.ClientIP,
        ServerIP: obj.ServerIP,
        ExitServerID: obj.ExitServerID,
        ManualDNS: obj.ManualDNS,
        IsCanPause: "IsCanPause" in obj ? obj.IsCanPause : null
      });

      requestGeoLookupAsync();
      break;

    case daemonResponses.DisconnectedResp:
      store.dispatch("vpnState/pauseState", PauseStateEnum.Resumed);
      store.commit(`vpnState/disconnected`, obj.ReasonDescription);
      if (store.state.settings.firewallDeactivateOnDisconnect === true) {
        await EnableFirewall(false);
      }
      requestGeoLookupAsync();
      break;

    case daemonResponses.ServerListResp:
      if (obj.VpnServers == null) break;
      store.dispatch(`vpnState/servers`, obj.VpnServers);
      break;
    case daemonResponses.PingServersResp:
      if (obj.PingResults == null) break;
      store.commit(`vpnState/serversPingStatus`, obj.PingResults);
      break;
    case daemonResponses.SetAlternateDNSResp:
      if (obj.IsSuccess == null || obj.IsSuccess !== true) break;
      if (obj.ChangedDNS == null) break;
      store.dispatch(`vpnState/dns`, obj.ChangedDNS);
      break;
    case daemonResponses.KillSwitchStatusResp:
      store.commit(`vpnState/firewallState`, obj);

      if (
        store.state.vpnState.firewallState.IsEnabled === false &&
        store.state.location == null &&
        store.state.vpnState.connectionState === VpnStateEnum.DISCONNECTED
      ) {
        // if FW disabled and no geolocation info - request geolocation
        requestGeoLookupAsync();
      }
      break;
    case daemonResponses.WiFiCurrentNetworkResp:
      store.commit(`vpnState/currentWiFiInfo`, {
        SSID: obj.SSID,
        IsInsecureNetwork: obj.IsInsecureNetwork
      });
      break;
    case daemonResponses.WiFiAvailableNetworksResp:
      store.commit(`vpnState/availableWiFiNetworks`, obj.Networks);
      break;

    case daemonResponses.ErrorResp:
      console.log("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!");
      console.log("!!!!!!!!!!!!!!!!!!!!!! ERROR RESP !!!!!!!!!!!!!!!!!!!!");
      console.log("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!");
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
function ConnectToDaemon(setConnState) {
  if (socket != null) {
    socket.destroy();
    socket = null;
  }

  if (setConnState === undefined)
    setConnState = function(state) {
      store.commit("daemonConnectionState", state);
    };

  return new Promise((resolve, reject) => {
    let connectF = function(err, portInfo) {
      if (err) {
        let errStr = `Unable to obtain IVPN daemon connection parameters: ${err}`;
        console.error("DAEMON CONNECTION ERROR: " + errStr);
        setConnState(DaemonConnectionType.NotConnected);
        reject(errStr);
        return;
      }

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
          // SEND HELLO
          // eslint-disable-next-line no-undef
          const secretBInt = BigInt(`0x${portInfo.secret}`);

          let appVersion = "";
          try {
            appVersion = `${require("electron").app.getVersion()}:Electron UI`;
          } catch (e) {
            console.error(e);
          }
          const helloReq = {
            Command: daemonRequests.Hello,
            Version: appVersion,
            Secret: secretBInt,
            GetServersList: true,
            GetStatus: true,
            GetConfigParams: true,
            KeepDaemonAlone: true
          };

          try {
            setConnState(DaemonConnectionType.Connecting);
            await sendRecv(helloReq, null, 10000);

            // the 'store.state.daemonVersion' and 'store.state.daemonIsOldVersionError' must be already initialized
            if (store.state.daemonIsOldVersionError === true) {
              setConnState(DaemonConnectionType.NotConnected);

              socket.destroy();
              socket = null;

              const err = new Error(
                `Unsupported IVPN Daemon version: v${store.state.daemonVersion} (minimum required v${config.MinRequiredDaemonVer})`
              );
              log.error(err);
              reject(err); // REJECT
              return;
            }

            // Saving 'connected' state to a daemon
            setConnState(DaemonConnectionType.Connected);

            // send logging + obfsproxy configuration
            SetLogging();
            SetObfsproxy();

            setTimeout(async () => {
              // Till this time we already must receive 'connected' info (if we are connected)
              // If we are in disconnected state and 'settings.autoConnectOnLaunch' enabled => start connection
              if (
                store.state.settings.autoConnectOnLaunch &&
                store.getters["vpnState/isDisconnected"]
              ) {
                console.log(
                  "Connecting on app start according to configuration (autoConnectOnLaunch)"
                );
                Connect();
              }
            }, 0);

            const pingRetryCount = 5;
            const pingTimeOutMs = 5000;
            PingServers(pingRetryCount, pingTimeOutMs);

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
      });

      socket.on("error", e => {
        log.error(`Connection error: ${e}`);
        reject(e);
      });

      log.debug("Connecting to daemon...");
      try {
        socket.connect(parseInt(portInfo.port, 10), "127.0.0.1");
      } catch (e) {
        console.error("Daemon connection error: ", e);
      }
    };

    getDaemonConnectionParams_cb(connectF);
  });
}

async function Login(accountID, force) {
  let resp = await sendRecv(
    {
      Command: daemonRequests.SessionNew,
      AccountID: accountID,
      ForceLogin: force
    },
    null,
    30000
  );

  if (resp.APIStatus === API_SUCCESS) commitSession(resp.Session);

  // Returning whole response object (even in case of error)
  // it contains details about error
  return resp;
}

async function Logout() {
  store.commit("settings/isExpectedAccountToBeLoggedIn", false);
  await KillSwitchSetIsPersistent(false);
  await EnableFirewall(false);
  await Disconnect();
  try {
    await sendRecv({
      Command: daemonRequests.SessionDelete
    });
  } catch (e) {
    console.error(e);
  }

  // It can happen that there will be no CONNECTION TO API or error on backend side
  // In this case the daemon will not logout.
  // Here we manually removing local session info
  commitNoSession();
}

async function AccountStatus() {
  return await sendRecv({ Command: daemonRequests.AccountStatus }, [
    daemonResponses.AccountStatusResp
  ]);
}

async function GetAppUpdateInfo() {
  try {
    let apiAlias = "";

    switch (Platform()) {
      case PlatformEnum.Windows:
        apiAlias = "updateInfo_Windows";
        break;
      case PlatformEnum.macOS:
        apiAlias = "updateInfo_macOS";
        break;
      case PlatformEnum.Linux:
        apiAlias = "updateInfo_Linux";
        break;
      default:
        throw new Error("Unsupported platform");
    }

    let resp = await sendRecv(
      { Command: daemonRequests.APIRequest, APIPath: apiAlias },
      [daemonResponses.APIResponse]
    );
    return JSON.parse(`${resp.ResponseData}`);
  } catch (e) {
    console.error("Failed to check latest update info: ", e);
  }
  return null;
}

async function GeoLookup(isRetryTry) {
  let isRealGeoLocationCheck = function() {
    return (
      store.state.vpnState.connectionState === VpnStateEnum.DISCONNECTED ||
      store.state.vpnState.pauseState === PauseStateEnum.Paused
    );
  };

  let retLocation = null;
  let isRealGeoLocationOnStart = isRealGeoLocationCheck();

  store.commit("isRequestingLocation", true); // mark 'Checking geolookup...'
  try {
    let resp = await sendRecv(
      { Command: daemonRequests.APIRequest, APIPath: "geo-lookup" },
      [daemonResponses.APIResponse]
    );

    if (resp.Error !== "")
      console.error("API 'geo-lookup' error: " + resp.Error);
    else {
      if (isRealGeoLocationOnStart != isRealGeoLocationCheck()) {
        log.error(
          `API ERROR: Unable to save geo-lookup result (connection state changed)`
        );
      } else {
        retLocation = JSON.parse(`${resp.ResponseData}`);
        if (retLocation == null || retLocation.ip_address === undefined) {
          log.error(`API ERROR:bad geo-lookup response`);
        } else {
          retLocation.isRealLocation = isRealGeoLocationOnStart;
          log.debug("API: 'geo-lookup' success.");
        }
      }
    }
  } catch (e) {
    console.error("geo-lookup error", e);
  }
  store.commit("location", retLocation);
  store.commit("isRequestingLocation", false); // un-mark 'Checking geolookup...'

  if (retLocation == null && !isRetryTry) {
    for (let r = 1; r <= 3; r++) {
      console.log(`Geo-lookup request failed. Retrying (${r})...`);

      let promise = new Promise(resolve => {
        store.commit("isRequestingLocation", true); // mark 'Checking geolookup...'
        setTimeout(() => resolve(GeoLookup(true)), r * 1000);
      });

      retLocation = await promise;
      if (retLocation != null) break;
    }
  }

  return retLocation;
}

let pingServersPromise = null;
async function PingServers(RetryCount, TimeOutMs) {
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
        RetryCount: RetryCount ? RetryCount : PingServersRetriesCnt,
        TimeOutMs: TimeOutMs ? TimeOutMs : PingServersTimeoutMs
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

async function GetDiagnosticLogs() {
  let logs = await sendRecv({ Command: daemonRequests.GenerateDiagnostics }, [
    daemonResponses.DiagnosticsGeneratedResp
  ]);

  // remove internal protocol variables
  delete logs.Command;
  delete logs.Idx;

  return logs;
}

async function Connect(entryServer, exitServer) {
  // if entryServer or exitServer is null -> will be used current selected servers
  // otherwise -> current selected servers will be replaced by a new values before connect

  let vpnParamsPropName = "";
  let vpnParamsObj = {};
  let settings = store.state.settings;

  // we are not in paused state anymore
  store.dispatch("vpnState/pauseState", PauseStateEnum.Resumed);

  store.commit("vpnState/connectionState", VpnStateEnum.CONNECTING);

  const isRandomExitSvr = store.getters["settings/isRandomExitServer"];

  // ENTRY SERVER
  if (entryServer != null) store.dispatch("settings/serverEntry", entryServer);
  else {
    if (store.getters["settings/isFastestServer"]) {
      // looking for fastest server
      let fastest = store.getters["vpnState/fastestServer"];
      if (fastest == null) {
        // request servers ping
        console.log(
          "Connect to fastest server (fastest server not defined). Pinging servers..."
        );
        await PingServers();
        fastest = store.getters["vpnState/fastestServer"];
      }
      if (fastest != null) store.dispatch("settings/serverEntry", fastest);
    } else if (store.getters["settings/isRandomServer"]) {
      // random server
      let servers = store.getters["vpnState/activeServers"];
      if (!isRandomExitSvr) {
        servers = servers.filter(
          s => s.country_code !== settings.serverExit.country_code
        );
      }
      let randomIdx = Math.floor(Math.random() * Math.floor(servers.length));
      store.dispatch("settings/serverEntry", servers[randomIdx]);
    }

    // EXIT SERVER
    if (exitServer != null) store.dispatch("settings/serverExit", exitServer);
    else if (isRandomExitSvr) {
      const servers = store.getters["vpnState/activeServers"];
      const exitServers = servers.filter(
        s => s.country_code !== settings.serverEntry.country_code
      );
      const randomIdx = Math.floor(
        Math.random() * Math.floor(exitServers.length)
      );
      store.dispatch("settings/serverExit", exitServers[randomIdx]);
    }
  }

  if (store.state.settings.firewallActivateOnConnect === true) {
    await EnableFirewall(true);
  }

  let port = store.getters["settings/getPort"];

  if (settings.vpnType === VpnTypeEnum.OpenVPN) {
    vpnParamsPropName = "OpenVpnParameters";

    vpnParamsObj = {
      EntryVpnServer: {
        ip_addresses: settings.serverEntry.ip_addresses
      },
      MultihopExitSrvID: settings.isMultiHop
        ? settings.serverExit.gateway.split(".")[0]
        : "",

      Port: {
        Port: port.port,
        Protocol: port.type // 0 === UDP
      }
    };

    const ProxyType = settings.ovpnProxyType;

    if (
      !isStrNullOrEmpty(ProxyType) &&
      !isStrNullOrEmpty(settings.ovpnProxyServer)
    ) {
      const ProxyPort = parseInt(settings.ovpnProxyPort);
      if (ProxyPort != null) {
        vpnParamsObj.ProxyType = ProxyType;
        vpnParamsObj.ProxyAddress = settings.ovpnProxyServer;
        vpnParamsObj.ProxyPort = ProxyPort;
        vpnParamsObj.ProxyUsername = settings.ovpnProxyUser;
        vpnParamsObj.ProxyPassword = settings.ovpnProxyPass;
      }
    }
  } else {
    vpnParamsPropName = "WireGuardParameters";
    vpnParamsObj = {
      EntryVpnServer: {
        Hosts: settings.serverEntry.hosts
      },

      Port: {
        Port: port.port
      }
    };
  }

  let currentDNS = "";
  if (settings.dnsIsCustom) currentDNS = settings.dnsCustom;
  if (settings.isAntitracker) {
    currentDNS = store.getters["vpnState/antitrackerIp"];
  }

  send({
    Command: daemonRequests.Connect,
    VpnType: settings.vpnType,
    [vpnParamsPropName]: vpnParamsObj,
    currentDNS
  });
}

async function Disconnect() {
  try {
    await ResumeConnection();
  } catch (e) {
    log.error(e);
  } finally {
    store.dispatch("vpnState/pauseState", PauseStateEnum.Resumed);
  }
  if (store.state.vpnState.connectionState === VpnStateEnum.CONNECTED)
    store.commit("vpnState/connectionState", VpnStateEnum.DISCONNECTING);
  await sendRecv(
    {
      Command: daemonRequests.Disconnect
    },
    [daemonResponses.DisconnectedResp]
  );
}

let isFirewallEnabledBeforePause = true;
async function PauseConnection(pauseSeconds) {
  if (pauseSeconds == null) return;
  if (store.state.vpnState.connectionState !== VpnStateEnum.CONNECTED) return;
  if (store.state.vpnState.pauseState === PauseStateEnum.Paused) return;

  store.dispatch("vpnState/pauseState", PauseStateEnum.Pausing);
  await sendRecv({
    Command: daemonRequests.PauseConnection
  });

  var pauseTill = new Date();
  pauseTill.setSeconds(pauseTill.getSeconds() + pauseSeconds);
  store.dispatch("uiState/pauseConnectionTill", pauseTill);

  try {
    // disable kill-switch (if not in firewall-persistent mode)
    if (!store.state.vpnState.firewallState.IsPersistent) {
      isFirewallEnabledBeforePause =
        store.state.vpnState.firewallState.IsEnabled;
      if (isFirewallEnabledBeforePause) await EnableFirewall(false);
    }
  } finally {
    store.dispatch("vpnState/pauseState", PauseStateEnum.Paused);
    requestGeoLookupAsync();
  }
}

async function ResumeConnection() {
  store.dispatch("uiState/pauseConnectionTill", null);

  if (store.state.vpnState.connectionState !== VpnStateEnum.CONNECTED) return;
  if (store.state.vpnState.pauseState === PauseStateEnum.Resumed) return;

  store.dispatch("vpnState/pauseState", PauseStateEnum.Resuming);
  await sendRecv({
    Command: daemonRequests.ResumeConnection
  });

  try {
    // switch back firewall into enabled state
    if (isFirewallEnabledBeforePause) await EnableFirewall(true);
  } finally {
    store.dispatch("vpnState/pauseState", PauseStateEnum.Resumed);
    requestGeoLookupAsync();
  }
}

async function EnableFirewall(enable) {
  if (store.state.vpnState.firewallState.IsPersistent === true) {
    console.error("Not allowed to change firewall state in Persistent mode");
    return;
  }

  await sendRecv({
    Command: daemonRequests.KillSwitchSetEnabled,
    IsEnabled: enable
  });
}

async function KillSwitchGetStatus() {
  await sendRecv({
    Command: daemonRequests.KillSwitchGetStatus
  });
}

async function KillSwitchSetAllowLANMulticast(AllowLANMulticast) {
  const Synchronously = true;
  await sendRecv({
    Command: daemonRequests.KillSwitchSetAllowLANMulticast,
    AllowLANMulticast,
    Synchronously
  });
}
async function KillSwitchSetAllowLAN(AllowLAN) {
  const Synchronously = true;
  await sendRecv({
    Command: daemonRequests.KillSwitchSetAllowLAN,
    AllowLAN,
    Synchronously
  });
}
async function KillSwitchSetIsPersistent(IsPersistent) {
  await sendRecv({
    Command: daemonRequests.KillSwitchSetIsPersistent,
    IsPersistent
  });
}

async function SetDNS(antitrackerIsEnabled) {
  let DNS = "";
  if (store.state.settings.dnsIsCustom) DNS = store.state.settings.dnsCustom;

  if (antitrackerIsEnabled != null) {
    // save antitracker configuration
    store.commit("settings/isAntitracker", antitrackerIsEnabled);
  }

  if (store.state.settings.isAntitracker)
    DNS = store.getters["vpnState/antitrackerIp"];

  if (store.state.vpnState.connectionState === VpnStateEnum.DISCONNECTED) {
    // no sense to send DNS-change request in disconnected state
    return;
  }

  // send change-request
  await sendRecv({
    Command: daemonRequests.SetAlternateDns,
    DNS
  });
}

async function SetLogging() {
  const enable = store.state.settings.logging;
  const Key = "enable_logging";
  let Value = `${enable}`;

  await send({
    Command: daemonRequests.SetPreference,
    Key,
    Value
  });
}

async function SetObfsproxy() {
  const enable = store.state.settings.connectionUseObfsproxy;
  const Key = "enable_obfsproxy";
  let Value = `${enable}`;

  await send({
    Command: daemonRequests.SetPreference,
    Key,
    Value
  });
}

async function WgRegenerateKeys() {
  if (store.state.vpnState.connectionState !== VpnStateEnum.DISCONNECTED) {
    throw Error("Unable to generate WireGuard keys in connected state");
  }

  await sendRecv({
    Command: daemonRequests.WireGuardGenerateNewKeys
  });
}

async function WgSetKeysRotationInterval(intervalSec) {
  await sendRecv({
    Command: daemonRequests.WireGuardSetKeysRotationInterval,
    Interval: intervalSec
  });
}

async function GetWiFiAvailableNetworks() {
  await send({
    Command: daemonRequests.WiFiAvailableNetworks
  });
}

export default {
  ConnectToDaemon,

  GetDiagnosticLogs,

  Login,
  Logout,
  AccountStatus,

  GetAppUpdateInfo,

  GeoLookup,
  PingServers,
  KillSwitchGetStatus,
  Connect,
  Disconnect,
  PauseConnection,
  ResumeConnection,

  EnableFirewall,
  KillSwitchSetAllowLANMulticast,
  KillSwitchSetAllowLAN,
  KillSwitchSetIsPersistent,

  SetDNS,
  SetLogging,
  SetObfsproxy,
  WgRegenerateKeys,
  WgSetKeysRotationInterval,

  GetWiFiAvailableNetworks
};
