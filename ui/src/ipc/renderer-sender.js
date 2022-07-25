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

const { ipcRenderer } = require("electron");

async function invoke(channel, ...args) {
  try {
    return await ipcRenderer.invoke(channel, ...args);
  } catch (e) {
    console.error(`(renderer-sender) ` + e);

    // remove prefix error text
    // (like: Error occurred in handler for 'renderer-request-login': Error: ...)
    const regexp = new RegExp(`^.+ '${channel}': (Error:)*`);
    const errStr = `${e}`;
    const corrected = errStr.replace(regexp, "");
    return new Promise((resolve, reject) => {
      reject(corrected.trim());
    });
  }
}

export default {
  // This object is using to expose ipcRenderer functionality (send\receive) only for limited channels
  // E.g. 'shared-mutation' plugin for vuex is requiring IPC communication
  GetSafeIpcRenderer() {
    // NOTE! in case of extending functionality,
    // do not forget to ensure that 'allowedChannels' contains correct (required) channels
    return {
      send: function (channel, ...args) {
        const allowedChannels = [
          "vuex-mutations-notify-main",
          "vuex-mutations-connect",
        ];

        if (allowedChannels.includes(channel)) {
          //console.log("SafeIpcRenderer send:", channel);
          return ipcRenderer.send(channel, ...args);
        } else
          console.log(
            "[ERROR] SafeIpcRenderer: unsupported channel to for 'send()' operation: ",
            channel
          );
      },
      on: function (channel, listener) {
        const allowedChannels = [
          "main-change-view-request",
          "vuex-mutations-notify-renderers",
        ];

        if (allowedChannels.includes(channel)) {
          //console.log("SafeIpcRenderer register event handler:", channel);
          ipcRenderer.on(channel, listener);
        } else
          console.log(
            "[ERROR] SafeIpcRenderer: unsupported channel to for registering event receiver: ",
            channel
          );
      },
    };
  },

  // After initialized, ask main thread about initial route
  GetInitRouteArgs: async function () {
    return await ipcRenderer.invoke("renderer-request-ui-initial-route-args");
  },

  // DAEMON

  ConnectToDaemon: async () => {
    return await invoke("renderer-request-connect-to-daemon");
  },
  RefreshStorage: () => {
    // function using to re-apply all mutations
    // This is required to send to renderer processes current storage state
    return invoke("renderer-request-refresh-storage");
  },

  Login: async (accountID, force, captchaID, captcha, confirmation2FA) => {
    return await invoke(
      "renderer-request-login",
      accountID,
      force,
      captchaID,
      captcha,
      confirmation2FA
    );
  },
  Logout: async (
    needToResetSettings,
    needToDisableFirewall,
    isCanDeleteSessionLocally
  ) => {
    return await invoke(
      "renderer-request-logout",
      needToResetSettings,
      needToDisableFirewall,
      isCanDeleteSessionLocally
    );
  },
  AccountStatus: async () => {
    return await invoke("renderer-request-account-status");
  },
  PingServers: () => {
    return invoke("renderer-request-ping-servers");
  },
  UpdateServersRequest: () => {
    return invoke("renderer-request-update-servers-request");
  },
  Connect: async () => {
    return await invoke("renderer-request-connect");
  },
  Disconnect: async () => {
    return await invoke("renderer-request-disconnect");
  },

  PauseConnection: async (pauseSeconds) => {
    if (pauseSeconds == null) return;
    //var pauseTill = new Date();
    //pauseTill.setSeconds(pauseTill.getSeconds() + pauseSeconds);
    return await invoke("renderer-request-pause-connection", pauseSeconds);
  },
  ResumeConnection: async () => {
    return await invoke("renderer-request-resume-connection");
  },

  EnableFirewall: async (isEnable) => {
    return await invoke("renderer-request-firewall", isEnable);
  },

  KillSwitchSetAllowApiServers: async (isEnable) => {
    return await invoke(
      "renderer-request-KillSwitchSetAllowApiServers",
      isEnable
    );
  },
  KillSwitchSetAllowLANMulticast: async (isEnable) => {
    return await invoke(
      "renderer-request-KillSwitchSetAllowLANMulticast",
      isEnable
    );
  },
  KillSwitchSetAllowLAN: async (isEnable) => {
    return await invoke("renderer-request-KillSwitchSetAllowLAN", isEnable);
  },
  KillSwitchSetIsPersistent: async (isEnable) => {
    return await invoke("renderer-request-KillSwitchSetIsPersistent", isEnable);
  },
  KillSwitchSetUserExceptions: async (userExceptions) => {
    return await invoke(
      "renderer-request-KillSwitchSetUserExceptions",
      userExceptions
    );
  },

  SplitTunnelGetStatus: async () => {
    return await invoke("renderer-request-SplitTunnelGetStatus");
  },
  SplitTunnelSetConfig: async (enabled, doReset) => {
    return await invoke(
      "renderer-request-SplitTunnelSetConfig",
      enabled,
      doReset
    );
  },
  SplitTunnelAddApp: async (execCmd) => {
    return await invoke("renderer-request-SplitTunnelAddApp", execCmd);
  },
  SplitTunnelRemoveApp: async (pid, execCmd) => {
    return await invoke("renderer-request-SplitTunnelRemoveApp", pid, execCmd);
  },

  GetInstalledApps: async () => {
    return await invoke("renderer-request-GetInstalledApps");
  },

  SetAutoconnectOnLaunch: async (isEnabled) => {
    return await invoke("renderer-request-SetAutoconnectOnLaunch", isEnabled);
  },
  SetLogging: async () => {
    return await invoke("renderer-request-set-logging");
  },
  SetObfsproxy: async () => {
    return await invoke("renderer-request-set-obfsproxy");
  },

  SetDNS: async (antitrackerIsEnabled) => {
    return await invoke("renderer-request-set-dns", antitrackerIsEnabled);
  },

  RequestDnsPredefinedConfigs: async () => {
    return await invoke("renderer-request-RequestDnsPredefinedConfigs");
  },

  GeoLookup: async () => {
    return await invoke("renderer-request-geolookup");
  },

  WgRegenerateKeys: async () => {
    return await invoke("renderer-request-wg-regenerate-keys");
  },
  WgSetKeysRotationInterval: async (intervalSec) => {
    return await invoke(
      "renderer-request-wg-set-keys-rotation-interval",
      intervalSec
    );
  },

  GetWiFiAvailableNetworks: async () => {
    return await invoke("renderer-request-wifi-get-available-networks");
  },

  // Diagnostic reports
  IsAbleToSendDiagnosticReport: () => {
    return ipcRenderer.sendSync("renderer-request-is-can-send-diagnostic-logs");
  },
  GetDiagnosticLogs: async () => {
    return await invoke("renderer-request-get-diagnostic-logs");
  },
  SubmitDiagnosticLogs: async (comment, data) => {
    return invoke("renderer-request-submit-diagnostic-logs", comment, data);
  },

  // UPDATES
  UpdateWindowClose: () => {
    return ipcRenderer.invoke("renderer-request-update-wnd-close");
  },
  UpdateWindowResizeContent: (width, height) => {
    return ipcRenderer.invoke(
      "renderer-request-update-wnd-resize",
      width,
      height
    );
  },
  AppUpdatesIsAbleToUpdate: () => {
    return ipcRenderer.sendSync(
      "renderer-request-app-updates-is-able-to-update"
    );
  },
  AppUpdatesCheck: async () => {
    return invoke("renderer-request-app-updates-check");
  },
  AppUpdatesUpgrade: async () => {
    return invoke("renderer-request-app-updates-upgrade");
  },
  AppUpdatesCancelDownload: async () => {
    return invoke("renderer-request-app-updates-cancel-download");
  },
  AppUpdatesInstall: async () => {
    return invoke("renderer-request-app-updates-install");
  },

  // AUTO-LAUNCH
  AutoLaunchIsEnabled: async () => {
    return invoke("renderer-request-auto-launch-is-enabled");
  },
  AutoLaunchSet: async (isEnabled) => {
    return invoke("renderer-request-auto-launch-set", isEnabled);
  },

  // COLOR SCHEME
  ColorScheme: () => {
    return ipcRenderer.sendSync("renderer-request-ui-color-scheme-get");
  },
  ColorSchemeSet: (scheme) => {
    return invoke("renderer-request-ui-color-scheme-set", scheme);
  },

  // NAVIGATION
  ShowAccountSettings: function () {
    ipcRenderer.send("renderer-request-show-settings-account");
  },
  ShowSettings: function () {
    ipcRenderer.send("renderer-request-show-settings-general");
  },
  ShowConnectionSettings: function () {
    ipcRenderer.send("renderer-request-show-settings-connection");
  },
  ShowWifiSettings: function () {
    ipcRenderer.send("renderer-request-show-settings-networks");
  },

  // CONTEXT MENU
  ShowContextMenuCopy: function () {
    ipcRenderer.send("renderer-request-show-context-menu-copy");
  },
  ShowContextMenuEdit: function () {
    ipcRenderer.send("renderer-request-show-context-menu-edit");
  },

  // DIALOG
  showMessageBoxSync: (diagConfig) => {
    return ipcRenderer.sendSync("renderer-request-showmsgboxsync", diagConfig);
  },
  showMessageBox: (diagConfig, doNotAttachToWindow) => {
    return invoke(
      "renderer-request-showmsgbox",
      diagConfig,
      doNotAttachToWindow
    );
  },

  showOpenDialogSync: (options) => {
    return ipcRenderer.sendSync("renderer-request-showOpenDialogSync", options);
  },
  showOpenDialog: (options) => {
    return invoke("renderer-request-showOpenDialog", options);
  },

  showModalDialog: async (
    dialogTypeName /*ModalDialogType*/,
    ownerWnd /*OwnerWindowType*/,
    windowConfig /*(nullable) BrowserWindow options*/
  ) => {
    return await invoke(
      "renderer-request-showModalDialog",
      dialogTypeName,
      ownerWnd,
      windowConfig
    );
  },

  // WINDOW
  closeCurrentWindow: () => {
    return invoke("renderer-request-close-current-window");
  },
  minimizeCurrentWindow: () => {
    return invoke("renderer-request-minimize-current-window");
  },
  getCurrentWindowProperties: () => {
    return ipcRenderer.sendSync("renderer-request-properties-current-window");
  },
  uiMinimize: (isMinimize) => {
    return invoke("renderer-request-UI-minimize", isMinimize);
  },

  // SHELL
  shellShowItemInFolder: (file) => {
    return invoke("renderer-request-shell-show-item-in-folder", file);
  },
  shellOpenExternal: (uri) => {
    return invoke("renderer-request-shell-open-external", uri);
  },

  // OS
  osRelease: () => {
    return ipcRenderer.sendSync("renderer-request-os-release");
  },
  Platform: () => {
    return ipcRenderer.sendSync("renderer-request-platform");
  },

  // APP
  appGetVersion: () => {
    return ipcRenderer.sendSync("renderer-request-app-getversion");
  },

  // HELPERS
  getAppIcon: (binaryPath) => {
    return invoke("renderer-request-getAppIcon", binaryPath);
  },

  // PARANOID MODE
  setParanoidModePassword: async (newPassword, oldPassword) => {
    return await invoke(
      "renderer-request-setParanoidModePassword",
      newPassword,
      oldPassword
    );
  },
  setLocalParanoidModePassword: async (password) => {
    return await invoke(
      "renderer-request-setLocalParanoidModePassword",
      password
    );
  },
};
