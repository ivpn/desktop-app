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

const api = require("@/api");
import { isStrNullOrEmpty } from "@/helpers/helpers";
import { API_SUCCESS } from "@/api/statuscode";
import { VpnTypeEnum, VpnStateEnum, PauseStateEnum } from "@/store/types";
import store from "@/store";

const DefaultResponseTimeoutMs = 15 * 1000;

// Socket to connect to a daemon
let socket = new net.Socket();
// Request number (increasing each new request)
let requestNo = 0;
// Array response waiters
const waiters = [];

const daemonRequests = Object.freeze({
  Hello: "Hello",
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
  WireGuardSetKeysRotationInterval: "WireGuardSetKeysRotationInterval"
});
const daemonResponses = Object.freeze({
  HelloResp: "HelloResp",
  VpnStateResp: "VpnStateResp",
  ConnectedResp: "ConnectedResp",
  DisconnectedResp: "DisconnectedResp",
  ServerListResp: "ServerListResp",
  PingServersResp: "PingServersResp",
  SetAlternateDNSResp: "SetAlternateDNSResp",
  KillSwitchStatusResp: "KillSwitchStatusResp",
  AccountStatusResp: "AccountStatusResp",
  ErrorResp: "ErrorResp"
});

// Read information about connection parameters from a file
function getDaemonConnectionParams() {
  let fpath = "";
  switch (os.platform()) {
    case "win32":
      // TODO: READ CORRECT PATH FROM REGISTRY
      fpath = "C:/Program Files/IVPN Client/etc/port.txt";
      break;
    case "darwin":
      fpath = "/Library/Application Support/IVPN/port.txt";
      break;
    case "linux":
      fpath = "/opt/ivpn/mutable/port.txt";
      break;
    default:
      throw new Error(`Not supported platform: '${os.platform()}'`);
  }

  const connData = fs.readFileSync(fpath).toString();

  const parsed = connData.split(":");
  if (parsed.length !== 2) {
    throw new Error("Failed to parse port-info file");
  }

  return { port: parsed[0], secret: parsed[1] };
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
  log.debug(`==> ${request.Command}`);
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
            reject(new Error("Response timeout"));
            break;
          }
        }
      },
      typeof (timeoutMs !== "undefined") && timeoutMs > 0
        ? timeoutMs
        : DefaultResponseTimeoutMs
    );
  });

  // register new waiter
  waiters.push(waiter);

  // send data
  send(request, requestNo);

  return promise;
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
  return session;
}

async function processResponse(response) {
  const obj = JSON.parse(response);

  if (obj != null && obj.Command != null) {
    // TODO: Full logging is only for debug. Must be removed from production!
    log.log(`<== ${obj.Command} ${response.length > 512 ? " ..." : response}`);
    // log.log(`<== ${response}`);
    //log.debug(`<== ${obj.Command}`);
  } else log.error(`<== ${response}`);

  if (obj == null || obj.Command == null || obj.Command.length <= 0) return;

  switch (obj.Command) {
    case daemonResponses.HelloResp:
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
      // forget about current location
      store.commit("location", null);

      store.dispatch(`vpnState/connectionInfo`, {
        VpnType: obj.VpnType,
        ConnectedSince: new Date(obj.TimeSecFrom1970 * 1000),
        ClientIP: obj.ClientIP,
        ServerIP: obj.ServerIP,
        ExitServerID: obj.ExitServerID,
        ManualDNS: obj.ManualDNS
      });
      break;

    case daemonResponses.DisconnectedResp:
      store.dispatch("vpnState/pauseState", PauseStateEnum.Resumed);
      store.commit(`vpnState/disconnected`, obj.ReasonDescription);
      if (store.state.settings.firewallOnOffOnConnect === true) {
        await EnableFirewall(false);
      }
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
        setTimeout(async () => {
          try {
            await api.default.GeoLookup();
          } catch (e) {
            console.log(e);
          }
        }, 0);
      }

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
      if (
        typeof waitingCommands !== "undefined" &&
        waitingCommands.length > 0
      ) {
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

function onDataReceived(received) {
  const responses = received.toString().split("\n");
  responses.forEach(j => {
    if (j.length > 0) {
      processResponse(j);
    }
  });
}

//////////////////////////////////////////////////////////////////////////////////////////
/// PUBLIC METHODS
//////////////////////////////////////////////////////////////////////////////////////////
async function ConnectToDaemon() {
  await new Promise((resolve, reject) => {
    // initialize current default state
    store.commit("vpnState/connectionState", VpnStateEnum.DISCONNECTED);
    store.commit("isDaemonConnected", false);

    if (socket != null) socket.destroy();

    socket = new net.Socket();
    const portInfo = getDaemonConnectionParams();
    socket.setNoDelay(true);

    let connTimeoutTimer;

    socket
      .on("connect", async () => {
        clearTimeout(connTimeoutTimer);

        // SEND HELLO
        // eslint-disable-next-line no-undef
        const secretBInt = BigInt(`0x${portInfo.secret}`);

        const helloReq = {
          Command: daemonRequests.Hello,
          Version: "0.1 UI2",
          Secret: secretBInt,
          GetServersList: true,
          GetStatus: true,
          GetConfigParams: true,
          KeepDaemonAlone: true
        };

        try {
          await sendRecv(helloReq, null, 10000);

          // Saving 'connected' state to a daemon
          store.commit("isDaemonConnected", true);

          // send logging + obfsproxy configuration
          SetLogging();
          SetObfsproxy();

          setTimeout(async () => {
            // request kill-switch status
            await KillSwitchGetStatus();

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

          send({
            Command: daemonRequests.PingServers,
            RetryCount: 5,
            TimeOutMs: 5000
          });

          resolve(); // RESOLVE
        } catch (e) {
          log.error(`Error receiving Hello response: ${e}`);
          reject(); // REJECT
          throw e;
        }
      })
      .on("data", onDataReceived);

    socket.on("close", () => {
      // Save 'disconnected' state
      store.commit("isDaemonConnected", false);
      socket.destroy();
      log.debug("Connection closed");
      reject(); // REJECT
    });

    socket.on("error", e => {
      log.error(`Connection error: ${e}`);
    });

    connTimeoutTimer = setTimeout(() => {
      socket.end();
      reject(); // REJECT
    }, 3000);

    log.debug("Connecting to daemon...");
    socket.connect(parseInt(portInfo.port, 10), "127.0.0.1");
  });
}

async function Login(accountID, force) {
  let resp = await sendRecv({
    Command: daemonRequests.SessionNew,
    AccountID: accountID,
    ForceLogin: force
  });

  if (resp.APIStatus === API_SUCCESS) commitSession(resp.Session);

  // Returning whole response object (even in case of error)
  // it contains details about error
  return resp;
}

async function Logout() {
  await KillSwitchSetIsPersistent(false);
  await EnableFirewall(false);
  await Disconnect();
  await sendRecv({
    Command: daemonRequests.SessionDelete
  });
}

async function AccountStatus() {
  return await sendRecv({ Command: daemonRequests.AccountStatus }, [
    daemonResponses.AccountStatusResp
  ]);
}

async function PingServers() {
  send({ Command: daemonRequests.PingServers, RetryCount: 3, TimeOutMs: 3000 });
}

async function Connect(entryServer, exitServer) {
  let vpnParamsPropName = "";
  let vpnParamsObj = {};

  if (store.state.settings.firewallOnOffOnConnect === true) {
    await EnableFirewall(true);
  }

  // we are not in paused state anymore
  store.dispatch("vpnState/pauseState", PauseStateEnum.Resumed);

  // if entryServer or exitServer is null -> will be used current selected servers
  // otherwise -> current selected servers will be replaced by a new values before connect
  if (entryServer != null) {
    store.dispatch("settings/serverEntry", entryServer);
  }
  if (exitServer != null) {
    store.dispatch("settings/serverExit", exitServer);
  }

  let settings = store.state.settings;
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

  store.commit("vpnState/connectionState", VpnStateEnum.CONNECTING);
  send({
    Command: daemonRequests.Connect,
    VpnType: settings.vpnType,
    [vpnParamsPropName]: vpnParamsObj,
    currentDNS
  });
}

async function Disconnect() {
  await ResumeConnection();
  store.dispatch("vpnState/pauseState", PauseStateEnum.Resumed);
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
async function PauseConnection() {
  if (store.state.vpnState.connectionState !== VpnStateEnum.CONNECTED) return;
  if (store.state.vpnState.pauseState === PauseStateEnum.Paused) return;

  store.dispatch("vpnState/pauseState", PauseStateEnum.Pausing);
  await sendRecv({
    Command: daemonRequests.PauseConnection
  });

  try {
    // disable kill-switch (if not in firewall-persistent mode)
    if (!store.state.vpnState.firewallState.IsPersistent) {
      isFirewallEnabledBeforePause =
        store.state.vpnState.firewallState.IsEnabled;
      if (isFirewallEnabledBeforePause) await EnableFirewall(false);
    }
  } finally {
    store.dispatch("vpnState/pauseState", PauseStateEnum.Paused);
  }
}

async function ResumeConnection() {
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

export default {
  ConnectToDaemon,
  Login,
  Logout,
  AccountStatus,
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
  WgSetKeysRotationInterval
};
