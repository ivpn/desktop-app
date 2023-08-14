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

import client from "@/daemon-client";
import config from "@/config";
import store from "@/store";
import { IsNewVersion } from "./helper";

export async function CheckUpdates() {
  try {
    let updates = await client.GetAppUpdateInfo();
    if (!updates || !updates.updateInfoRespRaw) return null;
    let updatesInfo = JSON.parse(`${updates.updateInfoRespRaw}`);
    if (!updatesInfo.daemon || !updatesInfo.daemon.version) return null;
    if (!updatesInfo.uiClient || !updatesInfo.uiClient.version) return null;
    return updatesInfo;
  } catch (e) {
    if (e instanceof SyntaxError)
      console.error("[updater] parsing update file info error: ", e.message);
    else console.error("[updater] error: ", e);

    return null;
  }
}

export function IsNewerVersion(updatesInfo, currDaemonVer, currUiVer) {
  return (
    IsNewVersion(currDaemonVer, updatesInfo.daemon.version) ||
    IsNewVersion(currUiVer, updatesInfo.uiClient.version)
  );
}

export function IsNeedSkipThisVersion(updatesInfo) {
  var settings = store.state.settings;

  return (
    settings.skipAppUpdate &&
    updatesInfo.daemon &&
    updatesInfo.uiClient &&
    settings.skipAppUpdate.daemonVersion &&
    settings.skipAppUpdate.uiVersion &&
    settings.skipAppUpdate.daemonVersion == updatesInfo.daemon.version &&
    settings.skipAppUpdate.uiVersion == updatesInfo.uiClient.version
  );
}

export function Upgrade(latestVersionInfo) {
  if (!latestVersionInfo) {
    console.error("Upgrade skipped: no information about latest version");
    return null;
  }

  try {
    require("electron").shell.openExternal(config.URLApps);
    return true; // trust that update is done. We can close 'update' window
  } catch (e) {
    console.error(e);
    return false;
  }
}
