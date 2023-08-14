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

import {
  SentrySendDiagnosticReport,
  SentryIsAbleToUse,
} from "@/sentry/sentry.js";

import { ipcMain, nativeTheme, dialog, app, shell } from "electron";
import path from "path";
import { Platform } from "@/platform/platform";
import { GetLinuxSnapEnvVars } from "@/helpers/main_platform";

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
  async (
    event,
    needToResetSettings,
    needToDisableFirewall,
    isCanDeleteSessionLocally
  ) => {
    return await client.Logout(
      needToResetSettings,
      needToDisableFirewall,
      isCanDeleteSessionLocally
    );
  }
);

ipcMain.handle("renderer-request-account-status", async () => {
  return await client.AccountStatus();
});

ipcMain.handle("renderer-request-ping-servers", async () => {
  return client.PingServers();
});

ipcMain.handle("renderer-request-update-servers-request", async () => {
  return client.ServersUpdateRequest();
});

ipcMain.handle("renderer-request-connect", async () => {
  return await client.Connect();
});
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
  "renderer-request-KillSwitchSetUserExceptions",
  async (event, userExceptions) => {
    return await client.KillSwitchSetUserExceptions(userExceptions);
  }
);

ipcMain.handle("renderer-request-SplitTunnelGetStatus", async () => {
  return await client.SplitTunnelGetStatus();
});
ipcMain.handle(
  "renderer-request-SplitTunnelSetConfig",
  async (event, enabled, doReset) => {
    return await client.SplitTunnelSetConfig(enabled, doReset);
  }
);
ipcMain.handle("renderer-request-SplitTunnelAddApp", async (event, execCmd) => {
  let funcShowMessageBox = function (dlgConfig) {
    return dialog.showMessageBox(
      event.sender.getOwnerBrowserWindow(),
      dlgConfig
    );
  };
  return await client.SplitTunnelAddApp(execCmd, funcShowMessageBox);
});
ipcMain.handle(
  "renderer-request-SplitTunnelRemoveApp",
  async (event, pid, execCmd) => {
    return await client.SplitTunnelRemoveApp(pid, execCmd);
  }
);

ipcMain.handle("renderer-request-GetInstalledApps", async () => {
  return await client.GetInstalledApps();
});

ipcMain.handle("renderer-request-SetUserPrefs", async (event, userPrefs) => {
  return await client.SetUserPrefs(userPrefs);
});

ipcMain.handle(
  "renderer-request-SetAutoconnectOnLaunch",
  async (event, isEnabled, isApplicableByDaemonInBackground) => {
    return await client.SetAutoconnectOnLaunch(
      isEnabled,
      isApplicableByDaemonInBackground
    );
  }
);
ipcMain.handle("renderer-request-set-logging", async (event, enable) => {
  return await client.SetLogging(enable);
});
ipcMain.handle(
  "renderer-request-set-obfsproxy",
  async (event, obfsproxyVer, obfs4Iat) => {
    return await client.SetObfsproxy(obfsproxyVer, obfs4Iat);
  }
);

ipcMain.handle("renderer-request-set-dns", async () => {
  return await client.SetDNS();
});

ipcMain.handle("renderer-request-RequestDnsPredefinedConfigs", async () => {
  return await client.RequestDnsPredefinedConfigs();
});

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

ipcMain.handle(
  "renderer-request-wifi-set-settings",
  async (event, wifiParams) => {
    return await client.SetWiFiSettings(wifiParams);
  }
);

ipcMain.handle("renderer-request-wifi-get-available-networks", async () => {
  return await client.GetWiFiAvailableNetworks();
});

// Diagnostic reports
ipcMain.on("renderer-request-is-can-send-diagnostic-logs", (event) => {
  event.returnValue = SentryIsAbleToUse();
});
ipcMain.handle("renderer-request-get-diagnostic-logs", async () => {
  let data = await client.GetDiagnosticLogs();
  if (data == null) data = {};

  const s = store.state;

  //  version
  let daemonVer = s.daemonVersion;
  if (!daemonVer) daemonVer = "UNKNOWN";
  if (s.daemonProcessorArch) daemonVer += ` [${s.daemonProcessorArch}]`;

  const uiVersion = app.getVersion() + ` [${process.arch}]`;

  // disabled functions
  let disabledFunctions = [];
  try {
    for (var propName in s.disabledFunctions) {
      if (!propName || !s.disabledFunctions[propName]) continue;
      disabledFunctions.push(`${propName} (${s.disabledFunctions[propName]})`);
    }
  } catch (e) {
    disabledFunctions.push([`ERROR: ${e}`]);
  }

  // account info
  let accInfo = "";
  try {
    const acc = s.account;
    accInfo = `${acc.accountStatus.CurrentPlan} (${
      acc.accountStatus.Active ? "Active" : "NOT ACTIVE"
    })`;
    if (acc.session.WgPublicKey)
      accInfo += `; wgKeys=OK ${acc.session.WgKeyGenerated}`;
    else accInfo += "; wgKeys=EMPTY";
  } catch (e) {
    accInfo = `ERROR: ${e}`;
  }

  // last disconnection
  try {
    data[" LastDisconnectionReason"] = "";
    if (
      s.vpnState.disconnectedInfo &&
      s.vpnState.disconnectedInfo.ReasonDescription
    )
      data[" LastDisconnectionReason"] =
        s.vpnState.disconnectedInfo.ReasonDescription;
  } catch (e) {
    data[" LastDisconnectionReason"] = `ERROR: ${e}`;
  }

  data[" Account"] =
    `${s.account.session ? s.account.session.AccountID : "???"}; ` + accInfo;
  if (disabledFunctions.length > 0)
    data[" DisabledFunctions"] = disabledFunctions.join("; ");
  data[" Firewall"] = JSON.stringify(s.vpnState.firewallState, null, 2);
  data[" ParanoidMode"] = s.paranoidModeStatus.IsEnabled ? "On" : "Off";
  data[" SplitTunneling"] = s.vpnState.splitTunnelling.IsEnabled ? "On" : "Off";
  data[" ParanoidMode"] = s.paranoidModeStatus.IsEnabled ? "On" : "Off";
  data[" Version"] = `Daemon=${daemonVer}; UI=${uiVersion}`;
  data[" Settings"] = JSON.stringify(s.settings, null, 2);

  return data;
});
ipcMain.handle(
  "renderer-request-submit-diagnostic-logs",
  async (event, comment, dataObj) => {
    let accountID = "";
    if (store.state.account.session != null)
      accountID = store.state.account.session.AccountID;

    let buildExtraInfo = "";
    if (GetLinuxSnapEnvVars()) {
      buildExtraInfo = "SNAP environement";
    }

    return SentrySendDiagnosticReport(
      accountID,
      comment,
      dataObj,
      store.state.daemonVersion,
      buildExtraInfo
    );
  }
);

// UPDATES
ipcMain.on("renderer-request-app-updates-is-able-to-update", (event) => {
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
ipcMain.on("renderer-request-ui-color-scheme-get", (event) => {
  event.returnValue = nativeTheme.themeSource;
});
ipcMain.handle("renderer-request-ui-color-scheme-set", (event, theme) => {
  store.dispatch("settings/colorTheme", theme);
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
ipcMain.handle("renderer-request-close-current-window", async (event) => {
  return await event.sender.getOwnerBrowserWindow().close();
});
ipcMain.handle("renderer-request-minimize-current-window", async (event) => {
  return await event.sender.getOwnerBrowserWindow().minimize();
});

ipcMain.on("renderer-request-properties-current-window", (event) => {
  const wnd = event.sender.getOwnerBrowserWindow();
  let retVal = null;
  if (wnd)
    retVal = {
      closable: wnd.closable,
      maximizable: wnd.maximizable,
      minimizable: wnd.minimizable,
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
ipcMain.on("renderer-request-os-release", (event) => {
  event.returnValue = os.release();
});
ipcMain.on("renderer-request-platform", (event) => {
  event.returnValue = Platform();
});

// APP
ipcMain.on("renderer-request-app-getversion", (event) => {
  event.returnValue = {
    Version: app.getVersion(),
    ProcessorArch: process.arch,
  };
});

// HELPERS
ipcMain.handle("renderer-request-getAppIcon", (event, binaryPath) => {
  return client.GetAppIcon(binaryPath);
});

// PARANOID MODE

ipcMain.handle(
  "renderer-request-setParanoidModePassword",
  async (event, newPassword, oldPassword) => {
    return await client.SetParanoidModePassword(newPassword, oldPassword);
  }
);

ipcMain.handle(
  "renderer-request-setLocalParanoidModePassword",
  async (event, password) => {
    return await client.SetLocalParanoidModePassword(password);
  }
);
