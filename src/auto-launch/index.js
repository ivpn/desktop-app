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

import { app } from "electron";
import { Platform, PlatformEnum } from "@/platform/platform";

// initialize application auto-launcher
var AutoLaunch = require("auto-launch");
let launcherOptions = { name: "IVPN", isHidden: true }; // isHidden is in use by Windows and Linux implementation (see function: WasOpenedAtLogin())
var autoLauncher = null;

if (Platform() === PlatformEnum.Linux) {
  const fs = require("fs");
  const binaryPath = "/opt/ivpn/ui/bin/ivpn-ui";
  if (fs.existsSync(binaryPath)) launcherOptions.path = binaryPath;
  else launcherOptions = null;
}

if (launcherOptions != null) autoLauncher = new AutoLaunch(launcherOptions);

function AutoLaunchIsInitialized() {
  return autoLauncher != null;
}

export function WasOpenedAtLogin() {
  try {
    if (Platform() === PlatformEnum.macOS) {
      let loginSettings = app.getLoginItemSettings();
      return loginSettings.wasOpenedAtLogin;
    }
    return app.commandLine.hasSwitch("hidden");
  } catch {
    return false;
  }
}

export async function AutoLaunchIsEnabled() {
  if (!AutoLaunchIsInitialized()) return null;
  try {
    return await autoLauncher.isEnabled();
  } catch (err) {
    console.error("Error obtaining 'LaunchAtLogin' value: ", err);
    return null;
  }
}

export async function AutoLaunchSet(isEnabled) {
  if (!AutoLaunchIsInitialized()) return;
  if (isEnabled) await autoLauncher.enable();
  else await autoLauncher.disable();
}
