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

"use strict";

import {
  app,
  protocol,
  BrowserWindow,
  Menu,
  dialog,
  nativeImage,
  ipcMain,
  nativeTheme
} from "electron";
import {
  createProtocol,
  installVueDevtools
} from "vue-cli-plugin-electron-builder/lib";

import { SentryInit } from "./sentry/sentry.js";
SentryInit();

// start waiting for events from Renderer processes
import "./ipc/main-listener";

import store from "@/store";
import { DaemonConnectionType, ColorTheme } from "@/store/types";
import daemonClient from "./daemon-client";
import { InitTray } from "./tray";
import { InitPersistentSettings, SaveSettings } from "./settings-persistent";
import { InitConnectionResumer } from "./connection-resumer";
import { InitTrustedNetworks } from "./trusted-wifi";
import { IsWindowHasTitle } from "@/platform/platform";
import { Platform, PlatformEnum } from "@/platform/platform";
import config from "@/config";

import { StartUpdateChecker } from "@/app-updater";

const isDevelopment = process.env.NODE_ENV !== "production";

// Keep a global reference of the window object, if you don't, the window will
// be closed automatically when the JavaScript object is garbage collected.
let win;
let settingsWindow;
let isTrayInitialized = false;
let lastRouteArgs = null; // last route arguments (requested by renderer process when window initialized)

// Only one instance of application can be started
const gotTheLock = app.requestSingleInstanceLock();
if (!gotTheLock) {
  console.log("Another instance of application is running.");
  app.quit();
} else {
  app.on("second-instance", () => {
    // Someone tried to run a second instance, we should focus our window.
    menuOnShow();
  });
}

// main process requesting information about 'initial route' after window created
ipcMain.handle("renderer-request-ui-initial-route-args", () => {
  return lastRouteArgs;
});
ipcMain.on("renderer-request-show-settings-general", () => {
  menuOnPreferences();
});
ipcMain.on("renderer-request-show-settings-account", () => {
  menuOnAccount();
});
ipcMain.on("renderer-request-show-settings-connection", () => {
  showSettings("connection");
});
ipcMain.on("renderer-request-show-settings-networks", () => {
  showSettings("networks");
});

if (gotTheLock) {
  InitPersistentSettings();
  InitConnectionResumer();
  InitTrustedNetworks();
  connectToDaemon();

  // INIT COLOR SCHEME
  try {
    if (store.state.settings.colorTheme)
      nativeTheme.themeSource = store.state.settings.colorTheme;
  } catch (e) {
    console.error("Failed to set color scheme: ", e);
  }
  // Scheme must be registered before the app is ready
  protocol.registerSchemesAsPrivileged([
    { scheme: "app", privileges: { secure: true, standard: true } }
  ]);

  // suppress the default menu
  Menu.setApplicationMenu(null);
  if (process.env.IS_DEBUG) {
    // DEBUG: TESTING MENU
    const dockMenu = Menu.buildFromTemplate([
      {
        label: "TEST (development menu)",
        submenu: [
          {
            label: "Open development tools",
            click() {
              if (win !== null) win.webContents.openDevTools();
            }
          },
          {
            label: "Switch to test view",
            click() {
              if (win !== null)
                win.webContents.send("change-view-request", "/test");
            }
          },
          {
            label: "Switch to main view",
            click() {
              if (win !== null)
                win.webContents.send("change-view-request", "/");
            }
          }
        ]
      }
    ]);
    Menu.setApplicationMenu(dockMenu);
  }

  // Quit when all windows are closed.
  app.on("window-all-closed", () => {
    // if we are waiting to save settings - save it immediately
    SaveSettings();
    win = null;
    lastRouteArgs = null;

    // if Tray initialized - just to keep app in tray
    if (
      isTrayInitialized !== true ||
      store.state.settings.minimizeToTray !== true
    )
      app.quit();
  });

  app.on("activate", () => {
    // On macOS it's common to re-create a window in the app when the
    // dock icon is clicked and there are no other windows open.
    if (win === null) {
      createWindow();
    }
  });

  // This method will be called when Electron has finished
  // initialization and is ready to create browser windows.
  // Some APIs can only be used after this event occurs.
  app.on("ready", async () => {
    try {
      InitTray(menuOnShow, menuOnPreferences, menuOnAccount, menuOnVersion);
      isTrayInitialized = true;
    } catch (e) {
      console.error(e);
    }

    if (isDevelopment && !process.env.IS_TEST) {
      // Install Vue Devtools
      // Devtools extensions are broken in Electron 6.0.0 and greater
      // See https://github.com/nklayman/vue-cli-plugin-electron-builder/issues/378 for more info
      // Electron will not launch with Devtools extensions installed on Windows 10 with dark mode
      // If you are not using Windows 10 dark mode, you may uncomment these lines
      // In addition, if the linked issue is closed, you can upgrade electron and uncomment these lines
      try {
        await installVueDevtools();
      } catch (e) {
        console.error("Vue Devtools failed to install:", e.toString());
      }
    }

    createWindow();
  });

  app.on("before-quit", async event => {
    // if disconnected -> close application immediately
    if (store.getters["vpnState/isDisconnected"]) {
      if (
        store.state.vpnState.firewallState.IsPersistent == false &&
        store.state.vpnState.firewallState.IsEnabled == true
      ) {
        let actionNo = dialog.showMessageBoxSync({
          type: "question",
          message: "Deactivate Firewall?",
          detail:
            "The IVPN Firewall is active.\nDo you want to deactivate it before exiting the application?",
          buttons: [
            "Cancel",
            "Keep Firewall activated & Quit",
            "Deactivate Firewall & Quit"
          ]
        });

        switch (actionNo) {
          case 0: // Cancel
            // discard exiting
            event.preventDefault();
            // show app window back (if not minimizing to tray)
            if (store.state.settings.minimizeToTray !== true)
              setTimeout(menuOnShow, 0);
            break;

          case 1: // Keep Firewall activate & Quit
            // do nothing here
            break;

          case 2: // Deactivate Firewall & Quit
            await daemonClient.EnableFirewall(false);
            break;
        }
      }
      return;
    }

    let actionNo = 0;
    if (store.state.settings.quitWithoutConfirmation) {
      actionNo = store.state.settings.disconnectOnQuit ? 0 : 1;
    } else {
      let msgBoxConfig = {
        type: "question",
        message: "Are you sure you want to quit?",
        detail: "You are connected to the VPN.",
        buttons: [
          "Cancel",
          "Disconnect VPN & Quit"
          //"Keep VPN connected & Quit"
        ]
      };

      if (win == null) actionNo = dialog.showMessageBoxSync(msgBoxConfig);
      else actionNo = dialog.showMessageBoxSync(win, msgBoxConfig);
    }
    switch (actionNo) {
      case 0: // Cancel
        // discard exiting
        event.preventDefault();

        // show app window back (if not minimizing to tray)
        if (store.state.settings.minimizeToTray !== true)
          setTimeout(menuOnShow, 0);
        break;

      case 1: // Exit & Disconnect VPN
        // Quit application only after connection closed
        event.preventDefault();
        setTimeout(async () => {
          try {
            if (
              store.state.settings.firewallDeactivateOnDisconnect ||
              store.state.settings.disconnectOnQuit
            )
              await daemonClient.EnableFirewall(false);

            await daemonClient.Disconnect();
          } catch (e) {
            console.log(e);
          } finally {
            app.quit(); // QUIT anyway
          }
        }, 0);
        break;

      //case 2: // Exit & Keep VPN connection
      //  // just close application
      //  break;
    }
  });

  // Exit cleanly on request from parent process in development mode.
  if (isDevelopment) {
    if (process.platform === "win32") {
      process.on("message", data => {
        if (data === "graceful-exit") {
          app.quit();
        }
      });
    } else {
      process.on("SIGTERM", () => {
        app.quit();
      });
    }
  }

  // subscribe to any changes in a store
  store.subscribe(mutation => {
    try {
      switch (mutation.type) {
        case "vpnState/currentWiFiInfo":
          // if wifi
          if (
            store.state.vpnState.currentWiFiInfo != null &&
            store.state.location == null
          )
            daemonClient.GeoLookup();
          break;
        case "settings/showAppInSystemDock":
          updateAppDockVisibility();
          break;
        case "account/session":
          if (store.getters["account/isLoggedIn"] !== true) {
            closeSettingsWindow();

            if (
              store.state.daemonConnectionState ===
              DaemonConnectionType.Connected
            ) {
              // in case of logged-out - ERASE SETTINGS TO DEFAULT STATE
              // Save parameters values which should not me erased
              console.log(
                "Account is not logged-in: Erasing settings to default value"
              );
              const vpnType = store.state.settings.vpnType;
              const showAppInSystemDock =
                store.state.settings.showAppInSystemDock;

              // erase settings
              store.commit("settings/resetToDefaults");

              // Necessary to initialize selected VPN servers
              store.dispatch("settings/updateSelectedServers");

              // set default obfsproxy value on daemon's side
              daemonClient.SetObfsproxy();

              // restore some parameters
              store.commit("settings/vpnType", vpnType);
              store.commit("settings/showAppInSystemDock", showAppInSystemDock);
            }
          }
          break;
        case "settings/minimizedUI":
          if (!store.state.settings.minimizedUI) closeSettingsWindow();
          break;
        default:
      }
    } catch (e) {
      console.error("Error in store subscriber:", e);
    }
  });
}

function getWindowIcon() {
  try {
    // loading window icon only for Linux.
    // The reest platforms will use icon from application binary
    if (Platform() !== PlatformEnum.Linux) return null;
    // eslint-disable-next-line no-undef
    return nativeImage.createFromPath(__static + "/64x64.png");
  } catch (e) {
    console.error(e);
  }
  return null;
}

// CREATE WINDOW
function createWindow() {
  // Create the browser window.

  let titleBarStyle = "default";
  if (!IsWindowHasTitle()) titleBarStyle = "hidden"; //"hiddenInset";

  let windowConfig = {
    backgroundColor: getBackgroundColor(),
    show: false,

    width: store.state.settings.minimizedUI
      ? config.MinimizedUIWidth
      : config.MaximizedUIWidth,
    height: 600,

    resizable: false,
    fullscreenable: false,
    maximizable: false,
    skipTaskbar:
      store.state.settings.showAppInSystemDock !== false ? false : true,

    center: true,
    title: "IVPN",

    titleBarStyle: titleBarStyle,
    autoHideMenuBar: true,

    webPreferences: {
      enableRemoteModule: true,
      // Use pluginOptions.nodeIntegration, leave this alone
      // See nklayman.github.io/vue-cli-plugin-electron-builder/guide/security.html#node-integration for more info
      // nodeIntegration: process.env.ELECTRON_NODE_INTEGRATION
      nodeIntegration: true
    }
  };

  let icon = getWindowIcon();
  if (icon != null) windowConfig.icon = icon;

  win = new BrowserWindow(windowConfig);

  if (process.env.WEBPACK_DEV_SERVER_URL) {
    // Load the url of the dev server if in development mode
    win.loadURL(process.env.WEBPACK_DEV_SERVER_URL);
    //if (!process.env.IS_TEST) win.webContents.openDevTools();
  } else {
    createProtocol("app");
    // Load the index.html when not in development
    win.loadURL("app://./index.html");
  }

  // show\hide app from system dock
  updateAppDockVisibility();

  win.once("ready-to-show", () => {
    win.show();
  });

  win.on("closed", () => {
    win = null;
  });
}

function createSettingsWindow(viewName) {
  if (win == null) createWindow();

  if (settingsWindow != null) {
    closeSettingsWindow();
  }
  if (viewName == null) viewName = "general";

  let windowConfig = {
    backgroundColor: getBackgroundColor(),
    show: false,

    width: 800,
    height: 600,

    resizable: false,
    fullscreenable: false,
    maximizable: false,

    parent: win,

    center: true,
    title: "Settings",

    autoHideMenuBar: true,

    webPreferences: {
      enableRemoteModule: true,
      nodeIntegration: true
    }
  };

  let icon = getWindowIcon();
  if (icon != null) windowConfig.icon = icon;

  settingsWindow = new BrowserWindow(windowConfig);

  if (process.env.WEBPACK_DEV_SERVER_URL) {
    // Load the url of the dev server if in development mode
    settingsWindow.loadURL(
      process.env.WEBPACK_DEV_SERVER_URL + `/#/settings/${viewName}`
    );
  } else {
    createProtocol("app");
    // Load the index.html when not in development
    settingsWindow.loadURL("app://./index.html" + `/#/settings/${viewName}`);
  }

  settingsWindow.once("ready-to-show", () => {
    settingsWindow.show();
  });
  settingsWindow.on("closed", () => {
    settingsWindow = null;
  });
}

function closeSettingsWindow() {
  if (settingsWindow == null) return;
  settingsWindow.destroy(); // close();
}

// INITIALIZE CONNECTION TO A DAEMON
async function connectToDaemon() {
  try {
    await daemonClient.ConnectToDaemon();
    // initialize app updater
    StartUpdateChecker(OnAppUpdateAvailable);
  } catch (e) {
    console.error("Failed to connect to IVPN Daemon: ", e);
  }
}

function showSettings(settingsViewName) {
  try {
    if (store.state.settings.minimizedUI) {
      createSettingsWindow(settingsViewName);
      return;
    }

    //menuOnShow();
    if (win !== null) {
      lastRouteArgs = {
        name: "settings",
        params: { view: settingsViewName }
      };

      // Temporary navigate to '\'. This is required only if we already showing 'settings' view
      // (to be able to re-init 'settings' view with new parameters)
      win.webContents.send("change-view-request", "/");
      win.webContents.send("change-view-request", lastRouteArgs);
    }
  } catch (e) {
    console.log(e);
  }
}

// show\hide app from SYSTEM DOCK
function updateAppDockVisibility() {
  if (store.state.settings.showAppInSystemDock) {
    // macOS
    if (app != null && app.dock != null) app.dock.show();

    // Windows
    if (win != null) {
      win.setSkipTaskbar(false);
    }
  } else {
    // macOS
    if (app != null && app.dock != null) app.dock.hide(); // remove from dock

    // Windows & Linux
    if (win != null) {
      win.setSkipTaskbar(true);
    }
  }
}

// MENU ITEMS
function menuOnShow() {
  try {
    if (!win) createWindow();
    else {
      win.restore();
      win.show();
      win.focus();
    }
  } catch (e) {
    console.error(e);
  }
}
function menuOnAccount() {
  menuOnShow();
  showSettings("account");
}
function menuOnPreferences() {
  menuOnShow();
  showSettings("general");
}
function menuOnVersion() {
  menuOnShow();
  showSettings("version");
}

// UPDATE
function OnAppUpdateAvailable(newDaemonVer, newUiVer) {
  // Arguments: updatesInfo.daemon.version, updatesInfo.uiClient.version, currDaemonVer, currUiVer
  try {
    // check if we should skip user notification for this version
    newDaemonVer = newDaemonVer.trim();
    newUiVer = newUiVer.trim();
    if (
      store.state.settings.skipAppUpdate &&
      store.state.settings.skipAppUpdate.daemonVersion &&
      store.state.settings.skipAppUpdate.uiVersion &&
      newDaemonVer === store.state.settings.skipAppUpdate.daemonVersion &&
      newUiVer === store.state.settings.skipAppUpdate.uiVersion
    ) {
      return;
    }

    // notify user
    if (win == null) createWindow();

    let msgBoxConfig = {
      type: "info",
      message: "A new version of IVPN Client is available!",
      buttons: ["Details ...", "Skip This Version", "Remind Me Later"]
    };

    const actionNo = dialog.showMessageBoxSync(win, msgBoxConfig);
    switch (actionNo) {
      case 0: // Details
        menuOnVersion();
        break;
      case 1: // Skip This Version
        store.commit("settings/skipAppUpdate", {
          daemonVersion: newDaemonVer,
          uiVersion: newUiVer
        });
        break;
      case 3: // Remind Me Later
        break;
    }
  } catch (e) {
    console.error(e);
  }
}

// COLORS
function getBackgroundColor() {
  // NOTE! the return values should be synchronized with CSS configuration
  // (src/components/scss/constants.scss)
  const theme = nativeTheme.themeSource;
  if (
    (theme === ColorTheme.system && nativeTheme.shouldUseDarkColors === true) ||
    theme === ColorTheme.dark
  )
    return "#1c1c1e";

  return "#FFFFFF";
}
