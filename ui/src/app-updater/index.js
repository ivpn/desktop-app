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
import { AppUpdateStage } from "@/store/types";
import { Platform, PlatformEnum } from "@/platform/platform";
const NoUpdaterErrorMessage = "App updater not available for this platform";

function getUpdater() {
  try {
    // Can be loaded different updaters according to platform
    if (IsGenericUpdater()) return require("./updater_generic");
    return require("./updater_linux");
  } catch (e) {
    console.error("[ERROR] getUpdater:", e);
  }
  return null;
}

function setState(updateState) {
  store.commit("uiState/appUpdateProgress", updateState);
}

function forgetVersionToSkip() {
  store.commit("settings/skipAppUpdate", null);
}

export function IsGenericUpdater() {
  if (Platform() === PlatformEnum.Linux) return false;
  return true;
}

export function IsAbleToCheckUpdate() {
  const updater = getUpdater();
  if (updater == null) return false;
  return true;
}

export function IsNewerVersion(updatesInfo, currDaemonVer, currUiVer) {
  const updater = getUpdater();
  if (updater == null) {
    console.warn(NoUpdaterErrorMessage);
    return false;
  }
  return updater.IsNewerVersion(updatesInfo, currDaemonVer, currUiVer);
}

export function StartUpdateChecker(onHasUpdateCallback) {
  if (!onHasUpdateCallback) {
    console.warn("Unable to start update checker: callback not defined");
    return false;
  }

  const updater = getUpdater();
  if (updater == null) {
    console.warn(NoUpdaterErrorMessage);
    return false;
  }

  try {
    const currDaemonVer = store.state.daemonVersion;
    const currUiVer = require("electron").app.getVersion();
    if (!currDaemonVer || !currUiVer) {
      console.warn("Unable to start update checker: app versions undefined");
      return false;
    }

    const doCheck = async function () {
      const isAutomaticCheck = true;
      const updatesInfo = await CheckUpdates(isAutomaticCheck);
      if (!updatesInfo) return;
      try {
        if (updater.IsNewerVersion(updatesInfo, currDaemonVer, currUiVer)) {
          if (updater.IsNeedSkipThisVersion(updatesInfo)) {
            return;
          }

          onHasUpdateCallback(updatesInfo, currDaemonVer, currUiVer);
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

export async function CheckUpdates(isAutomaticCheck) {
  try {
    console.log("Checking for app updates...");

    setState({
      state: AppUpdateStage.CheckingForUpdates,
    });

    if (isAutomaticCheck != true) forgetVersionToSkip();

    const updater = getUpdater();
    if (updater == null) throw "App updater not available for this platform";

    const settingsUpdates = store.state.settings.updates;

    let updatesInfo = await updater.CheckUpdates(
      isAutomaticCheck,
      settingsUpdates ? settingsUpdates.isBetaProgram : null
    );
    store.commit("latestVersionInfo", updatesInfo);

    setState({
      state: AppUpdateStage.CheckingFinished,
    });

    return updatesInfo;
  } catch (e) {
    console.error(e);
    setState({
      state: AppUpdateStage.Error,
      error: e,
    });
  }

  return null;
}

export function Upgrade() {
  const updater = getUpdater();
  if (updater == null) {
    console.warn(NoUpdaterErrorMessage);
    return null;
  }
  if (updater.Upgrade) return updater.Upgrade(store.state.latestVersionInfo);
  return null;
}

export function CancelDownload() {
  const updater = getUpdater();
  if (updater == null) {
    console.warn(NoUpdaterErrorMessage);
    return null;
  }
  if (updater.CancelDownload) return updater.CancelDownload();
}

export function Install() {
  const updater = getUpdater();
  if (updater == null) {
    console.warn(NoUpdaterErrorMessage);
    return null;
  }
  if (updater.Install) return updater.Install();
}
