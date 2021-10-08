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

import {
  SentrySendDiagnosticReport,
  SentryIsAbleToUse
} from "@/sentry/sentry.js";

import { ipcMain, nativeTheme, dialog, app, shell } from "electron";
import path from "path";
import { Platform } from "@/platform/platform";

import client from "../daemon-client";
import { CheckUpdates, IsAbleToCheckUpdate } from "@/app-updater";
import { AutoLaunchIsEnabled, AutoLaunchSet } from "@/auto-launch";
import store from "@/store";
import config from "@/config";
import { Upgrade, CancelDownload, Install } from "@/app-updater";

import os from "os";

// info: this event is processing in 'background.js'
//ipcMain.handle("renderer-request-connect-to-daemon", async () => {
//  return await client.ConnectToDaemon();
//});

ipcMain.handle("renderer-request-refresh-storage", async () => {
  // function using to re-apply all mutations
  // This is required to send to renderer processes current storage state
  store.commit("replaceState", store.state);
});

ipcMain.handle(
  "renderer-request-login",
  async (event, accountID, force, captchaID, captcha, confirmation2FA) => {
    return await client.Login(
      accountID,
      force,
      captchaID,
      captcha,
      confirmation2FA
    );
  }
);

ipcMain.handle(
  "renderer-request-logout",
  async (event, needToResetSettings, needToDisableFirewall) => {
    return await client.Logout(needToResetSettings, needToDisableFirewall);
  }
);

ipcMain.handle("renderer-request-account-status", async () => {
  return await client.AccountStatus();
});

ipcMain.handle("renderer-request-ping-servers", async () => {
  return client.PingServers();
});

ipcMain.handle(
  "renderer-request-connect",
  async (event, entryServer, exitServer) => {
    return await client.Connect(entryServer, exitServer);
  }
);
ipcMain.handle("renderer-request-disconnect", async () => {
  return await client.Disconnect();
});

ipcMain.handle(
  "renderer-request-pause-connection",
  async (event, pauseSeconds) => {
    return await client.PauseConnection(pauseSeconds);
  }
);
ipcMain.handle("renderer-request-resume-connection", async () => {
  return await client.ResumeConnection();
});

ipcMain.handle("renderer-request-firewall", async (event, enable) => {
  return await client.EnableFirewall(enable);
});
ipcMain.handle(
  "renderer-request-KillSwitchSetAllowApiServers",
  async (event, enable) => {
    return await client.KillSwitchSetAllowApiServers(enable);
  }
);
ipcMain.handle(
  "renderer-request-KillSwitchSetAllowLANMulticast",
  async (event, enable) => {
    return await client.KillSwitchSetAllowLANMulticast(enable);
  }
);
ipcMain.handle(
  "renderer-request-KillSwitchSetAllowLAN",
  async (event, enable) => {
    return await client.KillSwitchSetAllowLAN(enable);
  }
);
ipcMain.handle(
  "renderer-request-KillSwitchSetIsPersistent",
  async (event, enable) => {
    return await client.KillSwitchSetIsPersistent(enable);
  }
);

ipcMain.handle(
  "renderer-request-SplitTunnelSetConfig",
  async (event, enabled, apps) => {
    return await client.SplitTunnelSetConfig(enabled, apps);
  }
);
ipcMain.handle("renderer-request-GetInstalledApps", async () => {
  return await client.GetInstalledApps();
});

ipcMain.handle("renderer-request-set-logging", async () => {
  return await client.SetLogging();
});
ipcMain.handle("renderer-request-set-obfsproxy", async () => {
  return await client.SetObfsproxy();
});

ipcMain.handle(
  "renderer-request-set-dns",
  async (event, antitrackerIsEnabled) => {
    return await client.SetDNS(antitrackerIsEnabled);
  }
);

ipcMain.handle("renderer-request-geolookup", async () => {
  return await client.GeoLookup();
});

ipcMain.handle("renderer-request-wg-regenerate-keys", async () => {
  return await client.WgRegenerateKeys();
});

ipcMain.handle(
  "renderer-request-wg-set-keys-rotation-interval",
  async (event, intervalSec) => {
    return await client.WgSetKeysRotationInterval(intervalSec);
  }
);

ipcMain.handle("renderer-request-wifi-get-available-networks", async () => {
  return await client.GetWiFiAvailableNetworks();
});

// Diagnostic reports
ipcMain.on("renderer-request-is-can-send-diagnostic-logs", event => {
  event.returnValue = SentryIsAbleToUse();
});
ipcMain.handle("renderer-request-get-diagnostic-logs", async () => {
  let data = await client.GetDiagnosticLogs();
  if (data != null) {
    if (store.state.account.session != null)
      data[" IVPN User"] = store.state.account.session.AccountID;

    let daemonVer = store.state.daemonVersion;
    if (!daemonVer) daemonVer = "UNKNOWN";
    data[" DaemonVersion"] = daemonVer;

    data[" Settings"] = JSON.stringify(store.state.settings, null, 2);
  }
  return data;
});
ipcMain.handle(
  "renderer-request-submit-diagnostic-logs",
  async (event, comment, dataObj) => {
    let accountID = "";
    if (store.state.account.session != null)
      accountID = store.state.account.session.AccountID;
    return SentrySendDiagnosticReport(
      accountID,
      comment,
      dataObj,
      store.state.daemonVersion
    );
  }
);

// UPDATES
ipcMain.on("renderer-request-app-updates-is-able-to-update", event => {
  try {
    event.returnValue = IsAbleToCheckUpdate();
  } catch {
    event.returnValue = false;
  }
});
ipcMain.handle("renderer-request-app-updates-check", async () => {
  return await CheckUpdates();
});
ipcMain.handle("renderer-request-app-updates-upgrade", async () => {
  return await Upgrade();
});
ipcMain.handle("renderer-request-app-updates-cancel-download", async () => {
  return await CancelDownload();
});
ipcMain.handle("renderer-request-app-updates-install", async () => {
  return await Install();
});

// AUTO-LAUNCH
ipcMain.handle("renderer-request-auto-launch-is-enabled", async () => {
  return await AutoLaunchIsEnabled();
});
ipcMain.handle("renderer-request-auto-launch-set", async (event, isEnabled) => {
  return await AutoLaunchSet(isEnabled);
});

// COLOR SCHEME
ipcMain.on("renderer-request-ui-color-scheme-get", event => {
  event.returnValue = nativeTheme.themeSource;
});
ipcMain.handle("renderer-request-ui-color-scheme-set", (event, theme) => {
  store.dispatch("settings/colorTheme", theme);
  nativeTheme.themeSource = theme;
});

// DIALOG
ipcMain.on("renderer-request-showmsgboxsync", (event, diagConfig) => {
  event.returnValue = dialog.showMessageBoxSync(
    event.sender.getOwnerBrowserWindow(),
    diagConfig
  );
});
ipcMain.handle(
  "renderer-request-showmsgbox",
  async (event, diagConfig, doNotAttachToWindow) => {
    if (doNotAttachToWindow === true)
      return await dialog.showMessageBox(diagConfig);

    return await dialog.showMessageBox(
      event.sender.getOwnerBrowserWindow(),
      diagConfig
    );
  }
);

ipcMain.on("renderer-request-showOpenDialogSync", (event, options) => {
  event.returnValue = dialog.showOpenDialogSync(
    event.sender.getOwnerBrowserWindow(),
    options
  );
});
ipcMain.handle("renderer-request-showOpenDialog", async (event, options) => {
  return await dialog.showOpenDialog(
    event.sender.getOwnerBrowserWindow(),
    options
  );
});

// WINDOW
ipcMain.handle("renderer-request-UI-minimize", async (event, isMinimize) => {
  let win = event.sender.getOwnerBrowserWindow();
  if (win == null) return null;
  const animate = false;
  if (isMinimize)
    return await win.setBounds({ width: config.MinimizedUIWidth }, animate);
  else return await win.setBounds({ width: config.MaximizedUIWidth }, animate);
});
ipcMain.handle("renderer-request-close-current-window", async event => {
  return await event.sender.getOwnerBrowserWindow().close();
});
ipcMain.handle("renderer-request-minimize-current-window", async event => {
  return await event.sender.getOwnerBrowserWindow().minimize();
});

ipcMain.on("renderer-request-properties-current-window", event => {
  const wnd = event.sender.getOwnerBrowserWindow();
  let retVal = null;
  if (wnd)
    retVal = {
      closable: wnd.closable,
      maximizable: wnd.maximizable,
      minimizable: wnd.minimizable
    };
  event.returnValue = retVal;
});

// SHELL
ipcMain.handle(
  "renderer-request-shell-show-item-in-folder",
  async (event, file) => {
    file = path.normalize(file);
    return await shell.showItemInFolder(file);
  }
);
ipcMain.handle("renderer-request-shell-open-external", async (event, uri) => {
  if (uri == null) return;

  let isAllowedUrl = false;

  for (let p of config.URLsAllowedPrefixes) {
    if (uri == p || uri.startsWith(p + "/")) {
      isAllowedUrl = true;
      break;
    }
  }

  if (!isAllowedUrl) {
    const errMsg = `Opening the link '${uri}' is blocked. Not allowed to open links which are not starting from: ${config.URLsAllowedPrefixes}`;
    console.log(errMsg);
    throw Error(errMsg);
  }
  return shell.openExternal(uri);
});

// OS
ipcMain.on("renderer-request-os-release", event => {
  event.returnValue = os.release();
});
ipcMain.on("renderer-request-platform", event => {
  event.returnValue = Platform();
});

// APP
ipcMain.on("renderer-request-app-getversion", event => {
  event.returnValue = app.getVersion();
});

// HELPERS
ipcMain.handle("renderer-request-getAppIcon", (event, binaryPath) => {
  return client.GetAppIcon(binaryPath);
});
