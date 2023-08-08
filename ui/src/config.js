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
import path from "path";
function IsDebug() {
  if (process.env.WEBPACK_DEV_SERVER_URL) return true;
  return false;
}
function GetResourcesPath() {
  if (this.IsDebug()) {
    // eslint-disable-next-line no-undef
    return path.join(__static, "..", "extraResources");
  }
  return process.resourcesPath;
}

export default {
  MinRequiredDaemonVer: "3.11.15",

  MinimizedUIWidth: 320,
  MaximizedUIWidth: 800,
  UpdateWindowWidth: 600,

  // shellOpenExternal(...) allows only URLs started with this prefix
  URLsAllowedPrefixes: ["https://www.ivpn.net", "https://ivpn.net"],
  URLApps: "https://www.ivpn.net/apps/",

  IsDebug,
  GetResourcesPath,
};
