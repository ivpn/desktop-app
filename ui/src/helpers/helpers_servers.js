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

// CheckIsInaccessibleServer returns:
// - null if server is acceptble
// - object { sameGateway: true } - servers have same gateway
// - object { sameCountry: true } - servers are from same country (only if store.state.settings.multihopWarnSelectSameCountries === true)
// - objext { sameISP: true }     - servers are operated by same ISP (only if store.state.settings.multihopWarnSelectSameISPs === true)
export function CheckIsInaccessibleServer(isExitServer, server) {
  if (store == null || server == null) return null;
  if (store.state.settings.isMultiHop === false) return null;
  let ccSkip = "";
  let ispSkip = "";
  let gatewaySkip = false;

  let connected = !store.getters["vpnState/isDisconnected"];
  if (
    // ENTRY SERVER
    !isExitServer &&
    store.state.settings.serverExit &&
    (connected || !store.state.settings.isRandomExitServer)
  ) {
    ccSkip = store.state.settings.serverExit.country_code;
    ispSkip = store.state.settings.serverExit.isp;
    gatewaySkip = store.state.settings.serverExit.gateway;
  } else if (
    // EXIT SERVER
    isExitServer &&
    store.state.settings.serverEntry &&
    (connected || !store.state.settings.isRandomServer)
  ) {
    ccSkip = store.state.settings.serverEntry.country_code;
    ispSkip = store.state.settings.serverEntry.isp;
    gatewaySkip = store.state.settings.serverEntry.gateway;
  }

  if (server.gateway === gatewaySkip)
    return {
      sameGateway: true,
      message: "Entry and exit servers are the same",
      detail: "Please select a different entry or exit server.",
    };

  if (
    store.state.settings.multihopWarnSelectSameCountries === true &&
    server.country_code === ccSkip
  )
    return {
      sameCountry: true,
      message: "Entry and exit servers located in the same country",
      detail:
        "Using Multi-Hop servers from the same country may decrease your privacy.",
    };

  if (
    store.state.settings.multihopWarnSelectSameISPs === true &&
    server.isp === ispSkip
  )
    return {
      sameISP: true,
      message: "Entry and exit servers are operated by the same ISP",
      detail:
        "Using Multi-Hop servers operated by the same ISP may decrease your privacy.",
    };

  return null;
}

export async function CheckAndNotifyInaccessibleServer(isExitServer, server) {
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

  let svrInaccessibleInfo = CheckIsInaccessibleServer(isExitServer, server);
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
