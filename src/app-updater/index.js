//
//  UI for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app-ui2
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

import store from "@/store";

function getUpdater() {
  try {
    // Can be loaded different updaters according to platform (switch (Platform()) {...} )
    // Currently, we are using common updater for all platforms
    return require("./common_updater");
  } catch (e) {
    console.error("[ERROR] IsAbleToCheckUpdate :", e);
  }
  return null;
}

export function IsAbleToCheckUpdate() {
  const updater = getUpdater();
  if (updater == null) return false;
  return true;
}

export function StartUpdateChecker(onHasUpdateCallback) {
  const updater = getUpdater();
  if (updater == null) {
    console.warn("App updater not available for this platform");
    return false;
  }

  try {
    const currDaemonVer = store.state.daemonVersion;
    const currUiVer = require("electron").app.getVersion();
    if (!currDaemonVer || !currUiVer) {
      console.warn(
        "Unable to start app update checker: current app versions undefined"
      );
      return false;
    }

    const doCheck = async function() {
      const updatesInfo = await CheckUpdates();

      try {
        if (
          updatesInfo &&
          onHasUpdateCallback &&
          (IsNewerVersion(currDaemonVer, updatesInfo.daemon.version) ||
            IsNewerVersion(currUiVer, updatesInfo.uiClient.version))
        ) {
          onHasUpdateCallback(
            updatesInfo.daemon.version,
            updatesInfo.uiClient.version,
            currDaemonVer,
            currUiVer
          );
        }
      } catch (e) {
        console.error(e);
        return;
      }
    };
    // check for updates in 5 seconds after initialization
    setTimeout(doCheck, 1000 * 5);

    // start periodical update check
    setInterval(doCheck, 1000 * 60 * 60 * 12); // 12-hours interval
  } catch (e) {
    console.error(e);
    return false;
  }
  return true;
}

export async function CheckUpdates() {
  const updater = getUpdater();
  if (updater == null) {
    console.error("App updater not available for this platform");
    return null;
  }

  console.log("Checking for app updates...");
  try {
    let updatesInfo = await updater.CheckUpdates();
    if (!updatesInfo) return null;
    if (!updatesInfo.daemon || !updatesInfo.daemon.version) return null;
    if (!updatesInfo.uiClient || !updatesInfo.uiClient.version) return null;

    store.commit("latestVersionInfo", updatesInfo);
    return updatesInfo;
  } catch (e) {
    console.error(e);
  }
  return null;
}

export function Upgrade() {
  const updater = getUpdater();
  if (updater == null) {
    console.error("App updater not available for this platform");
    return null;
  }

  return updater.Upgrade(store.state.latestVersionInfo);
}

export function IsNewerVersion(oldVer, newVer) {
  if (!oldVer || !newVer) return false;
  oldVer = oldVer.trim();
  newVer = newVer.trim();
  if (!oldVer || !newVer) return false;

  const newVerStrings = newVer.split(".");
  const curVerStrings = oldVer.split(".");

  try {
    for (let i = 0; i < newVerStrings.length && i < curVerStrings.length; i++) {
      if (parseInt(newVerStrings[i], 10) > parseInt(curVerStrings[i], 10))
        return true;
      if (parseInt(newVerStrings[i], 10) < parseInt(curVerStrings[i], 10))
        return false;
    }

    if (newVerStrings.length > curVerStrings.length) {
      for (let i = curVerStrings.length; i < newVerStrings.length; i++) {
        if (parseInt(newVerStrings[i], 10) > 0) return true;
      }
    }
  } catch (e) {
    console.log(e);
  }
  return false;
}
