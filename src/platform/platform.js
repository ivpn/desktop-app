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
import { IsRenderer } from "../helpers/helpers";

export const PlatformEnum = Object.freeze({
  unknown: 0,
  macOS: 1,
  Linux: 2,
  Windows: 3
});

import os from "os";

export function Platform() {
  //return PlatformEnum.macOS;
  //return PlatformEnum.Linux;
  //return PlatformEnum.Windows;

  if (IsRenderer()) return window.ipcSender.Platform();

  // main process
  switch (os.platform()) {
    case "win32":
      return PlatformEnum.Windows;
    case "linux":
      return PlatformEnum.Linux;
    case "darwin":
      return PlatformEnum.macOS;
    default:
      return PlatformEnum.unknown;
  }
}

export function IsWindowHasTitle() {
  return Platform() !== PlatformEnum.macOS;
}
