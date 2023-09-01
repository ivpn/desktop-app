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

const fs = require("fs");
import merge from "deepmerge";
import path from "path";
import { app } from "electron";

import store from "@/store";

import { DnsEncryption } from "@/store/types";

var saveSettingsTimeout = null;
var saveAccStateTimeout = null;

const userDataFolder = app.getPath("userData");
const filename = path.join(userDataFolder, "ivpn-settings.json");
const filenameAccState = path.join(userDataFolder, "acc-state.json");

export function InitPersistentSettings() {
  // persistent SETTINGS
  if (fs.existsSync(filename)) {
    try {
      // merge data from a settings file
      const data = fs.readFileSync(filename);
      const settings = JSON.parse(data);

      // UPGRADING FROM OLD SETTINGS (v3.5.2 -> v3.6.1)
      if (settings.dnsCustom !== undefined) {
        settings.dnsCustomCfg = {
          DnsHost: settings.dnsCustom,
          Encryption: DnsEncryption.None,
          DohTemplate: "",
        };
        delete settings.dnsCustom;
      }

      // UPGRADING from OLD SETTINGS (from v3.10.0 and older)
      try {
        // only gateway ID in use for serversFavoriteList ("us-tx.wg.ivpn.net" => "us-tx")
        let favSvs = settings.serversFavoriteList.map((gw) => gw.split(".")[0]);
        settings.serversFavoriteList = favSvs;
      } catch (e) {
        console.error("InitPersistentSettings (serversFavoriteList upd.): ", e);
      }

      // UPGRADING from OLD SETTINGS (from v3.10.23 and v3.11.5)
      try {
        // changed location of obfsproxy configuration
        if (
          settings.daemonSettings &&
          settings.daemonSettings.ObfsproxyConfig
        ) {
          settings.openvpnObfsproxyConfig =
            settings.daemonSettings.ObfsproxyConfig;
          delete settings.daemonSettings.ObfsproxyConfig;
        }
      } catch (e) {
        console.error("InitPersistentSettings (obfsproxyConfig upd.): ", e);
      }

      // apply settings data
      const mergedState = merge(store.state.settings, settings, {
        arrayMerge: combineMerge,
      });
      store.commit("settings/replaceState", mergedState);
    } catch (e) {
      console.error(e);
    }
  } else {
    console.log(
      "Settings file not exist (probably, the first application start)"
    );
  }

  // ACCOUNT STATE
  if (fs.existsSync(filenameAccState)) {
    try {
      // merge data from a settings file
      const data = fs.readFileSync(filenameAccState);
      const accState = JSON.parse(data);

      if (accState.Active)
        store.commit("account/accountStatus", { Account: accState });
    } catch (e) {
      console.error(e);
    }
  } else {
    console.log("Account state file not exist (probably, not logged in)");
  }

  // STORE EVENT SUBSCRIPTION
  store.subscribe((mutation) => {
    try {
      // SETTINGS
      // saves settings object each 2 seconds
      // (in case if mutations happened in 'settings' named module)
      if (mutation.type.startsWith("settings/")) {
        if (saveSettingsTimeout != null) clearTimeout(saveSettingsTimeout);
        saveSettingsTimeout = setTimeout(() => {
          SaveSettings();
        }, 2000);
      }
      // ACCOUNT STATE
      else if (mutation.type.startsWith("account/")) {
        if (saveAccStateTimeout != null) clearTimeout(saveAccStateTimeout);
        saveAccStateTimeout = setTimeout(() => {
          SaveAccountState();
        }, 2000);
      }
    } catch (e) {
      console.error(
        `Error in InitPersistentSettings (store.subscribe ${mutation.type}):`,
        e
      );
    }
  });
}

export function SaveSettings() {
  if (saveSettingsTimeout == null) return;

  clearTimeout(saveSettingsTimeout);
  saveSettingsTimeout = null;

  try {
    let data = JSON.stringify(store.state.settings, null, 2);
    fs.writeFileSync(filename, data);
  } catch (e) {
    console.error("Failed to save settings:" + e);
  }
}

export function SaveAccountState() {
  if (saveAccStateTimeout == null) return;

  clearTimeout(saveAccStateTimeout);
  saveAccStateTimeout = null;

  try {
    if (
      store.getters["account/isLoggedIn"] !== true ||
      !store.state.account ||
      !store.state.account.accountStatus
    ) {
      if (fs.existsSync(filenameAccState)) fs.unlinkSync(filenameAccState);
    } else {
      let data = JSON.stringify(store.state.account.accountStatus, null, 2);
      fs.writeFileSync(filenameAccState, data);
    }
  } catch (e) {
    console.error("Failed to save account state:" + e);
  }
}

function combineMerge(target, source, options) {
  const emptyTarget = (value) => (Array.isArray(value) ? [] : {});
  const clone = (value, options) => merge(emptyTarget(value), value, options);
  const destination = target.slice();

  source.forEach(function (e, i) {
    if (typeof destination[i] === "undefined") {
      const cloneRequested = options.clone !== false;
      const shouldClone = cloneRequested && options.isMergeableObject(e);
      destination[i] = shouldClone ? clone(e, options) : e;
    } else if (options.isMergeableObject(e)) {
      destination[i] = merge(target[i], e, options);
    } else if (target.indexOf(e) === -1) {
      destination.push(e);
    }
  });

  return destination;
}
