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

import store from "@/store";
import { IsRenderer } from "@/helpers/helpers";

export function IsServerSupportIPv6(server) {
  if (!server) return null;
  if (!server.hosts) return null;

  for (let h of server.hosts) {
    if (h && h.ipv6 && h.ipv6.local_ip) return true;
  }
  return false;
}

// Each server's host has an ISP property, so this function
// returns an array of unique ISPs 
// NOTE: we should not use server.isp property anymore!
export function GetSreverISPs(server) {
  if (!server || !server.hosts) return [];

  const isps = new Set();
  for (const host of server.hosts) {
    if (host && host.isp) {
      isps.add(host.isp);
    }
  }
  return Array.from(isps);
}

// CheckIsInaccessibleServer returns:
// - null if server is acceptble
// - object { sameGateway: true } - servers have same gateway
// - object { sameCountry: true } - servers are from same country (only if store.state.settings.multihopWarnSelectSameCountries === true)
// - objext { sameISP: true }     - servers are operated by same ISP (only if store.state.settings.multihopWarnSelectSameISPs === true)
export function CheckIsInaccessibleServer(isExitServer, server, host = null) {
  const settings = store.state.settings;
  if (!settings) return null;

  if (store == null || server == null) return null;
  if (settings.isMultiHop === false) return null;
    
  let getSingleSvrISP = function (server) {
    if (!server || !server.hosts) return null;
    const isps = GetSreverISPs(server);
    if (isps.length === 1) {
      return isps[0];
    }
    return null; // Multiple ISPs, cannot determine a single ISP
  }

  // if ispCheck is not provided, means we can skip ISP check
  // since the server has multiple ISPs
  let ispCheck = (host)? host.isp : getSingleSvrISP(server);

  let ccSkip = "";
  let ispSkip = "";
  let gatewaySkip = false;

  let connected = !store.getters["vpnState/isDisconnected"];
  if (
    // ENTRY SERVER
    !isExitServer &&
    settings.serverExit &&
    (connected || !settings.isRandomExitServer)
  ) {
    ccSkip = settings.serverExit.country_code;
    gatewaySkip = settings.serverExit.gateway;    
    const selectedExitHost = store.getters["settings/selectedExitHost"];
    ispSkip = (selectedExitHost)? selectedExitHost.isp : getSingleSvrISP(settings.serverExit);

  } else if (
    // EXIT SERVER
    isExitServer &&
    settings.serverEntry &&
    (connected || !settings.isRandomServer)
  ) {
    ccSkip = settings.serverEntry.country_code;    
    gatewaySkip = settings.serverEntry.gateway;
    const selectedEntryHost = store.getters["settings/selectedEntryHost"];
    ispSkip = (selectedEntryHost)? selectedEntryHost.isp : getSingleSvrISP(settings.serverEntry);
  }

  if (server.gateway === gatewaySkip)
    return {
      sameGateway: true,
      message: "Entry and exit servers are the same",
      detail: "Please select a different entry or exit server.",
    };

  if (
    settings.multihopWarnSelectSameCountries === true &&
    server.country_code === ccSkip
  )
    return {
      sameCountry: true,
      message: "Entry and exit servers located in the same country",
      detail:
        "Using Multi-Hop servers from the same country may decrease your privacy.",
    };

  if (
    settings.multihopWarnSelectSameISPs === true &&
    ispCheck && ispSkip && ispCheck === ispSkip
  )
    return {
      sameISP: true,
      message: "Entry and exit servers are operated by the same ISP",
      detail:
        "Using Multi-Hop servers operated by the same ISP may decrease your privacy.",
    };

  return null;
}

export async function CheckAndNotifyInaccessibleServer(isExitServer, server, host = null) {
  let showMessageBoxFunc = null;
  if (IsRenderer()) {
    // renderer
    const sender = window.ipcSender;
    showMessageBoxFunc = sender.showMessageBox;
  } else {
    // background
    const { dialog } = require("electron");
    showMessageBoxFunc = dialog.showMessageBox;
  }

  if (!showMessageBoxFunc) {
    console.error(
      "CheckAndNotifyInaccessibleServer: showMessageBoxFunc not initialised",
    );
    return true;
  }

  let svrInaccessibleInfo = CheckIsInaccessibleServer(isExitServer, server, host);
  if (svrInaccessibleInfo !== null) {
    if (svrInaccessibleInfo.sameGateway === true) {
      await showMessageBoxFunc({
        type: "info",
        buttons: ["OK"],
        message: svrInaccessibleInfo.message,
        detail: svrInaccessibleInfo.detail,
      });
      return false;
    }

    if (
      svrInaccessibleInfo.sameCountry === true ||
      svrInaccessibleInfo.sameISP === true
    ) {
      let ret = await showMessageBoxFunc({
        type: "warning",
        buttons: ["Continue", "Cancel"],
        message: svrInaccessibleInfo.message,
        detail: svrInaccessibleInfo.detail,
      });
      if (ret.response == 1) return false; // cancel
    }
  }
  return true;
}
