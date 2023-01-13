//
//  UI for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
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

export function IsOsDarkColorScheme() {
  //matchMedia method not supported
  if (!window.matchMedia) return false;
  //OS theme setting detected as dark
  if (window.matchMedia("(prefers-color-scheme: dark)").matches) return true;
  return false;
}

//
// Allow input only numerical characters
//
// Usage example:
// <input ref="myInput" ...
//   mounted() {
//  SetInputFilterNumbers(this.$refs.myInput);
//
export function SetInputFilterNumbers(inputElement) {
  inputElement.addEventListener("keypress", function (evt) {
    try {
      var charCode = evt.which ? evt.which : evt.keyCode;
      if (charCode > 31 && (charCode < 48 || charCode > 57))
        evt.preventDefault();
    } catch (e) {
      console.error(e);
    }
  });
  inputElement.addEventListener("paste", function (evt) {
    try {
      const pastedData = evt.clipboardData.getData("text");
      const isOK = /^\d*$/.test(pastedData);
      if (!isOK) evt.preventDefault();
    } catch (e) {
      console.error(e);
    }
  });
}

export function GetTimeLeftText(endTime /*Date()*/) {
  if (endTime == null) return "";

  if (typeof endTime === "string" || endTime instanceof String)
    endTime = Date.parse(endTime);

  let secondsLeft = (endTime - new Date()) / 1000;
  if (secondsLeft <= 0) return "";

  function two(i) {
    if (i < 10) i = "0" + i;
    return i;
  }

  const h = Math.floor(secondsLeft / (60 * 60));
  const m = Math.floor((secondsLeft - h * 60 * 60) / 60);
  const s = Math.floor(secondsLeft - h * 60 * 60 - m * 60);
  return `${two(h)} : ${two(m)} : ${two(s)}`;
}

// CheckIsInaccessibleServer returns:
// - null if server is acceptble
// - object { sameGateway: true } - servers have same gateway
// - object { sameCountry: true } - servers are from same country (only if store.state.settings.multihopWarnSelectSameCountries === true)
// - objext { sameISP: true }     - servers are operated by same ISP (only if store.state.settings.multihopWarnSelectSameISPs === true)
export function CheckIsInaccessibleServer(store, isExitServer, server) {
  if (store == null || isExitServer == null || server == null) return null;
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
      message: "Entry and exit servers are identical",
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
        "Using Multi-Hop servers from different countries is good practice to increase anonymity.",
    };

  if (
    store.state.settings.multihopWarnSelectSameISPs === true &&
    server.isp === ispSkip
  )
    return {
      sameISP: true,
      message: "Entry and exit servers are operated by the same ISP",
      detail:
        "Using Multi-Hop servers operated by different ISPs is good practice to increase anonymity.",
    };

  return null;
}

const sender = window.ipcSender;
export async function CheckAndNotifyInaccessibleServer(
  store,
  isExitServer,
  server
) {
  let svrInaccessibleInfo = CheckIsInaccessibleServer(
    store,
    isExitServer,
    server
  );
  if (svrInaccessibleInfo !== null) {
    if (svrInaccessibleInfo.sameGateway === true) {
      await sender.showMessageBox({
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
      let ret = await sender.showMessageBox({
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
