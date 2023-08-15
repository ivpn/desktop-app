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
import { IsRenderer } from "../helpers/helpers";

export const PlatformEnum = Object.freeze({
  unknown: 0,
  macOS: 1,
  Linux: 2,
  Windows: 3,
});

import os from "os";

let hashedCurrPlatform = null;
export function Platform() {
  if (hashedCurrPlatform) return hashedCurrPlatform;

  if (IsRenderer()) hashedCurrPlatform = window.ipcSender.Platform();
  else {
    // main process
    switch (os.platform()) {
      case "win32":
        hashedCurrPlatform = PlatformEnum.Windows;
        break;
      case "linux":
        hashedCurrPlatform = PlatformEnum.Linux;
        break;
      case "darwin":
        hashedCurrPlatform = PlatformEnum.macOS;
        break;
      default:
        hashedCurrPlatform = PlatformEnum.unknown;
    }
  }
  return hashedCurrPlatform;
}

export function IsWindowHasFrame() {
  return Platform() === PlatformEnum.macOS;
}
