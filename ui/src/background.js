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

"use strict";

import {
  app,
  protocol,
  BrowserWindow,
  Menu,
  dialog,
  nativeImage,
  ipcMain,
  nativeTheme,
  screen,
  session,
  powerMonitor,
} from "electron";
import {
  createProtocol,
  //installVueDevtools
} from "vue-cli-plugin-electron-builder/lib";

import { SentryInit } from "./sentry/sentry.js";
SentryInit();

// start waiting for events from Renderer processes
import "./ipc/main-listener";

import store from "@/store";
import { AutoLaunchSet, AutoLaunchIsEnabled } from "@/auto-launch";
import { DaemonConnectionType, ColorTheme } from "@/store/types";
import daemonClient from "./daemon-client";
import darwinDaemonInstaller from "./daemon-client/darwin-installer";
import { InitTray } from "./tray";
import { InitPersistentSettings, SaveSettings } from "./settings-persistent";
import { IsWindowHasFrame } from "@/platform/platform";
import { Platform, PlatformEnum } from "@/platform/platform";
import config from "@/config";
import path from "path";

import { StartUpdateChecker, CheckUpdates } from "@/app-updater";
import { WasOpenedAtLogin } from "@/auto-launch";

// default copy/edit context menu event handlers
require("@/context-menu/main");

const isDevelopment = process.env.NODE_ENV !== "production";

// Keep a global reference of the window object, if you don't, the window will
// be closed automatically when the JavaScript object is garbage collected.
let win;
let settingsWindow;
let updateWindow;
let isAppReadyToQuit = false;

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
    console.log("The second app instance was tried to start.");
    menuOnShow();
  });
}

// abortController can be used to cancel active messageBox dialogs when app exiting.
// Example:
//      dialog.showMessageBox(win, { signal: abortController.signal, })
//      abortController.abort();
let abortController = new AbortController();
// Every time controller is aborted, we need to reinitialize new object (to become back in not-aborted state)
function abortControllerAbort() {
  abortController.abort();
  abortController = new AbortController();
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
ipcMain.on("renderer-request-show-settings-firewall", () => {
  showSettings("firewall");
});
ipcMain.on("renderer-request-show-settings-antitracker", () => {
  showSettings("antitracker");
});
ipcMain.on("renderer-request-show-settings-SplitTunnel", () => {
  showSettings("splittunnel");
});
ipcMain.handle("renderer-request-connect-to-daemon", async () => {
  return await connectToDaemon();
});
ipcMain.handle("renderer-request-update-wnd-close", async () => {
  if (!updateWindow) return;
  updateWindow.destroy();
});
ipcMain.handle(
  "renderer-request-update-wnd-resize",
  async (event, width, height) => {
    if (!updateWindow || (!width && !height)) return;
    if (!width) width = config.UpdateWindowWidth;
    if (!height) height = updateWindow.getContentSize()[1];
    updateWindow.setContentSize(width, height);
  }
);
ipcMain.handle("renderer-request-SplitTunnelAddApp", async (event, execCmd) => {
  LaunchAppInSplitTunnel(execCmd, event);
});

async function LaunchAppInSplitTunnel(execCmd, event) {
  let wnd = win;
  if (event && event.sender) wnd = event.sender.getOwnerBrowserWindow();

  let funcShowMessageBox = function (dlgConfig) {
    return dialog.showMessageBox(wnd, dlgConfig);
  };

  try {
    // manuall app ...
    if (!execCmd) {
      let dlgFilters = [];
      if (Platform() === PlatformEnum.Windows) {
        dlgFilters = [
          { name: "Executables", extensions: ["exe"] },
          { name: "All files", extensions: ["*"] },
        ];
      } else {
        dlgFilters = [{ name: "All files", extensions: ["*"] }];
      }
      let dlgConfig = {
        title: "Select application to launch",
        filters: dlgFilters,
        properties: ["openFile"],
      };
      var ret = dialog.showOpenDialogSync(wnd, dlgConfig);
      if (!ret || ret.canceled || ret.length == 0) return;
      execCmd = ret[0];
    }

    return await daemonClient.SplitTunnelAddApp(execCmd, funcShowMessageBox);
  } catch (e) {
    console.error(e);
    funcShowMessageBox({
      type: "error",
      buttons: ["OK"],
      detail: e.toString(),
      message: "Failed to launch application in Split Tunnel environment",
    });
    return;
  }
}

if (gotTheLock) {
  InitPersistentSettings();
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
    { scheme: "app", privileges: { secure: true, standard: true } },
  ]);

  const isMac = process.platform === "darwin";
  const template = [
    // { role: 'appMenu' }
    ...(isMac
      ? [
          {
            label: app.name,
            submenu: [
              { type: "separator" },
              { role: "hide" },
              { role: "hideothers" },
              { role: "unhide" },
              { type: "separator" },
              { role: "quit" },
            ],
          },
        ]
      : []),
    // { role: 'fileMenu' }
    {
      label: "File",
      submenu: [isMac ? { role: "close" } : { role: "quit" }],
    },
    // { role: 'windowMenu' }
    {
      label: "Window",
      submenu: [
        { role: "minimize" },
        ...(isMac
          ? [
              { type: "separator" },
              { role: "front" },
              { type: "separator" },
              { role: "window" },
            ]
          : [{ role: "close" }]),
      ],
    },
    {
      role: "help",
      submenu: [
        {
          label: "Learn More",
          click: async () => {
            const { shell } = require("electron");
            await shell.openExternal("https://www.ivpn.net/knowledgebase");
          },
        },
      ],
    },
  ];
  if (process.env.IS_DEBUG) {
    // DEBUG: TESTING MENU
    template.push({
      label: "TEST (dev. menu)",
      submenu: [
        {
          label: "Open development tools",
          click() {
            if (win !== null) win.webContents.openDevTools();
            if (updateWindow !== null) updateWindow.webContents.openDevTools();
          },
        },
        {
          label: "Switch to test view",
          click() {
            if (win !== null)
              win.webContents.send("main-change-view-request", "/test");
          },
        },
        {
          label: "Switch to main view",
          click() {
            if (win !== null)
              win.webContents.send("main-change-view-request", "/");
          },
        },
      ],
    });
  }
  const menu = Menu.buildFromTemplate(template);
  Menu.setApplicationMenu(menu);

  // This method will be called when Electron has finished
  // initialization and is ready to create browser windows.
  // Some APIs can only be used after this event occurs.
  app.on("ready", async () => {
    daemonClient.RegisterMsgBoxFunc(dialog.showMessageBox);

    // MACOS: Check is application is located in correct place (path)
    if (Platform() === PlatformEnum.macOS && !process.env.IS_DEBUG) {
      let appPath = app.getAppPath();
      if (!appPath.startsWith("/Applications/IVPN.app/")) {
        console.log(`Failed to start. Wrong application path: ${appPath}`);

        dialog.showMessageBoxSync({
          type: "error",
          message: "Unable to start IVPN Client",
          detail:
            "IVPN client can only run from the Applications folder. Please move the IVPN.app into the /Applications folder",
          buttons: ["Quit"],
        });

        console.log(`Exiting ...`);
        app.quit();
        return;
      }
    }

    // Deny all permission requests
    // https://www.electronjs.org/docs/latest/tutorial/security#5-handle-session-permission-requests-from-remote-content
    session.defaultSession.setPermissionRequestHandler(
      (webContents, permission, callback) => {
        console.log("Permission request blocked: ", permission);
        callback(false);
      }
    );

    try {
      InitTray(
        menuOnShow,
        menuOnPreferences,
        menuOnAccount,
        menuOnCheckUpdates,
        LaunchAppInSplitTunnel
      );
      isTrayInitialized = true;
    } catch (e) {
      console.error(e);
    }

    /*
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
    }*/

    if (store.state.settings.minimizeToTray && WasOpenedAtLogin()) {
      // do not show main application window when application was started automatically on login
      // (if enabled minimizeToTray)
      const doNotShowWhenReady = true;
      createWindow(doNotShowWhenReady);
    } else {
      createWindow();
    }

    if (isDevelopment) {
      try {
        win.webContents.openDevTools();
      } catch (e) {
        console.error("Failed to open dev tools:", e.toString());
      }
    }
  });

  // Disable navigation + Disable creation of new windows
  app.on("web-contents-created", (event, contents) => {
    // Disable navigation
    // https://www.electronjs.org/docs/latest/tutorial/security#13-disable-or-limit-navigation
    contents.on("will-navigate", (event, navigationUrl) => {
      console.log("[WARNING] Preventing navigation to:", navigationUrl);
      event.preventDefault();
    });

    // Disable creation of new windows
    // https://www.electronjs.org/docs/latest/tutorial/security#14-disable-or-limit-creation-of-new-windows
    contents.setWindowOpenHandler(({ url }) => {
      console.log("[WARNING] Preventing creating new window:", url);
      return { action: "deny" };
    });
  });

  app.on("activate", () => {
    menuOnShow();
  });

  // Quit when all windows are closed.
  app.on("window-all-closed", async () => {
    lastRouteArgs = null;

    if (
      isAppReadyToQuit != true &&
      isTrayInitialized == true &&
      store.state.settings.minimizeToTray == true
    )
      return; // skip quit (stay in tray)

    // the app 'before-quit' event will be raised
    app.quit();
  });

  // Event: 'shutdown' (Linux and macOS only !!!)
  // Emitted when the system is about to reboot or shut down.
  powerMonitor.on("shutdown", () => {
    isAppReadyToQuit = true;
  });

  app.on("before-quit", async (event) => {
    // save last window position in order to be able to restore it
    if (win) store.commit("settings/windowRestorePosition", win.getBounds());
    // if we are waiting to save settings - save it immediately
    SaveSettings();

    if (isAppReadyToQuit == true) return; // quit

    // discard exiting
    event.preventDefault();
    if ((await isCanQuit()) == true) {
      isAppReadyToQuit = true;
      app.quit();
    }
  });

  // Exit cleanly on request from parent process in development mode.
  if (isDevelopment) {
    if (process.platform === "win32") {
      process.on("message", (data) => {
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
  store.subscribe((mutation) => {
    try {
      switch (mutation.type) {
        case "settings/resetToDefaults":
          try {
            updateAppDockVisibility();
            nativeTheme.themeSource = store.state.settings.colorTheme;
            AutoLaunchSet(false);
            applyMinimizedState();
          } catch (e) {
            console.debug("Failed to reset settings to defaults: " + e);
          }
          break;

        case "settings/colorTheme":
          nativeTheme.themeSource = store.state.settings.colorTheme;
          break;

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

        case "settings/daemonSettings":
          setTimeout(async () => {
            try {
              let dSettings = store.state.settings.daemonSettings;
              if (
                dSettings.IsAutoconnectOnLaunchDaemon === true ||
                dSettings.WiFi.canApplyInBackground === true
              ) {
                if (AutoLaunchIsEnabled() !== true) {
                  console.log(
                    "Background VPN management is active: Enabling 'Launch at login' ..."
                  );
                  AutoLaunchSet(true);
                }
              }
            } catch (e) {
              console.error(e);
            }
          }, 0);
          break;

        case "account/session":
          if (store.getters["account/isLoggedIn"] !== true) {
            closeSettingsWindow();
          }
          break;
        case "settings/minimizedUI":
          if (!store.state.settings.minimizedUI) closeSettingsWindow();
          applyMinimizedState();
          break;

        case "account/accountStatus":
          // When IVPN apps detect a plan downgrade (from Pro to Standard), an active VPN connection that uses Pro features (MultiHop or Port forwarding)
          // should be disconnected or reconnected with Standard plan features.
          // Before the active VPN connection is disconnected by the app,
          // a UI alert should be presented with the option to reconnect without pro features (e.g. SingleHop instead of MultiHop).
          if (store.getters["vpnState/isConnected"] === true) {
            if (
              store.state.settings.isMultiHop === true &&
              store.getters["account/isMultihopAllowed"] !== true
            ) {
              let msgBoxConfig = {
                type: "question",
                message: "Subscription is changed to IVPN Standard",
                detail:
                  "Active VPN connection is using Pro plan features (MultiHop or Port forwarding) and will be disconnected.",
                buttons: ["OK", "Reconnect with SingleHop VPN"],
                signal: abortController.signal, // cancel dialog on window close
              };
              setTimeout(async () => {
                let action = null;
                if (win == null)
                  action = await dialog.showMessageBox(msgBoxConfig);
                else action = await dialog.showMessageBox(win, msgBoxConfig);

                switch (action.response) {
                  case 0: // OK
                    daemonClient.Disconnect();
                    break;

                  case 1: // Reconnect with SingleHop VPN
                    daemonClient.Disconnect();
                    daemonClient.Connect();
                    break;
                }
              }, 0);
            }
          }
          break;

        default:
      }
    } catch (e) {
      console.error("Error in store subscriber:", e);
    }
  });
}

async function isCanQuit() {
  if (store.getters["vpnState/isInverseSplitTunnel"]) {
    // temporary enable application icon in system dock
    setAppDockVisibility(true);

    let msgBoxConfig = {
      type: "question",
      message: "Deactivate Split Tunnel?",
      detail:
        "The Inverse Split Tunnel mode is active.\nDo you want to deactivate Split Tunnel before exiting the application?",
      buttons: [
        "Cancel",
        "Keep Split Tunnel active",
        "Deactivate Split Tunnel",
      ],
    };

    let actionNo = 0;
    let action = null;
    if (win == null) action = await dialog.showMessageBox(msgBoxConfig);
    else action = await dialog.showMessageBox(win, msgBoxConfig);
    actionNo = action.response;

    switch (actionNo) {
      case 0: // Cancel
        return false;

      case 1: // Keep & Quit
        // do nothing here
        break;

      case 2: // Deactivate & Quit
        await daemonClient.SplitTunnelSetConfig(false);
        break;
    }
  }

  // if disconnected -> close application immediately
  if (store.getters["vpnState/isDisconnected"]) {
    if (
      store.state.vpnState.firewallState.IsPersistent == false &&
      store.state.vpnState.firewallState.IsEnabled == true
    ) {
      let msgBoxConfig = {
        type: "question",
        message: "Deactivate Firewall?",
        detail:
          "The IVPN Firewall is active.\nDo you want to deactivate it before exiting the application?",
        buttons: [
          "Cancel",
          "Keep Firewall activated and Quit",
          "Deactivate Firewall and Quit",
        ],
      };

      // temporary enable application icon in system dock
      setAppDockVisibility(true);

      let actionNo = 0;
      let action = null;
      if (win == null) action = await dialog.showMessageBox(msgBoxConfig);
      else action = await dialog.showMessageBox(win, msgBoxConfig);
      actionNo = action.response;

      switch (actionNo) {
        case 0: // Cancel
          return false;

        case 1: // Keep Firewall activate & Quit
          // do nothing here
          break;

        case 2: // Deactivate Firewall & Quit
          await daemonClient.EnableFirewall(false);
          break;
      }
    }
    return true;
  }

  let actionNo = 0;
  if (store.state.settings.quitWithoutConfirmation) {
    actionNo = 1;
  } else {
    let msgBoxConfig = {
      type: "question",
      message: "Are you sure you want to quit?",
      detail: "You are connected to the VPN.",
      buttons: ["Cancel", "Disconnect VPN and Quit"],
    };

    // temporary enable application icon in system dock
    setAppDockVisibility(true);

    // Using 'showMessageBox' not 'showMessageBoxSync' - this is required to not to block Tray menu items
    let action = null;
    if (win == null) action = await dialog.showMessageBox(msgBoxConfig);
    else action = await dialog.showMessageBox(win, msgBoxConfig);
    actionNo = action.response;

    // restore default visibility of the application icon in system dock
    updateAppDockVisibility();
  }

  switch (actionNo) {
    case 0: // Cancel
      return false;

    case 1: // Exit & Disconnect VPN
      // Quit application only after connection closed
      try {
        if (
          !store.state.settings.quitWithoutConfirmation ||
          store.state.settings.disconnectOnQuit
        ) {
          if (store.state.settings.firewallDeactivateOnDisconnect)
            await daemonClient.EnableFirewall(false);
          await daemonClient.Disconnect();
        }
      } catch (e) {
        console.log(e);
      }
      return true;
  }
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

function createBrowserWindow(config) {
  config.webPreferences = {
    preload: path.join(__dirname, "preload.js"),

    // Use pluginOptions.nodeIntegration, leave this alone
    // See nklayman.github.io/vue-cli-plugin-electron-builder/guide/security.html#node-integration for more info
    //nodeIntegration: process.env.ELECTRON_NODE_INTEGRATION
    nodeIntegration: false,
    contextIsolation: true,
    sandbox: true,
    "disableBlinkFeatures ": "Auxclick",
  };

  let icon = getWindowIcon();
  if (icon != null) config.icon = icon;

  // Note: the navigation and opening new windows is disabled for this window
  // For details, refer to definition (above): "app.on("web-contents-created",..."
  return new BrowserWindow(config);
}

// CREATE WINDOW
function createWindow(doNotShowWhenReady) {
  // Create the browser window.

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
      store.state.settings.showAppInSystemDock !== false ? false : true, // not applicable for Linux (since Electron v20)

    center: true,
    title: "IVPN",

    frame: IsWindowHasFrame(),
    titleBarStyle: "hidden", // applicable only for macOS
    autoHideMenuBar: true,
  };

  win = createBrowserWindow(windowConfig);

  // restore window position
  let lastPos = store.state.settings.windowRestorePosition;
  if (lastPos && lastPos.x && lastPos.y) {
    const displays = screen.getAllDisplays();
    let isWindowVisibleOnScreen = false;
    displays.forEach((display) => {
      if (
        lastPos.x > display.workArea.x &&
        lastPos.x + 50 < display.workArea.x + display.workArea.width &&
        lastPos.y > display.workArea.y &&
        lastPos.y + 50 < display.workArea.y + display.workArea.height
      )
        isWindowVisibleOnScreen = true;
    });

    if (isWindowVisibleOnScreen == true)
      win.setBounds({ x: lastPos.x, y: lastPos.y });
  }

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

  if (doNotShowWhenReady != true) {
    win.once("ready-to-show", () => {
      win.show();
    });
  }

  win.on("close", async (event) => {
    // save last window position in order to be able to restore it
    if (win) store.commit("settings/windowRestorePosition", win.getBounds());

    if (isAppReadyToQuit == true) return;
    if (
      isTrayInitialized == true &&
      store.state.settings.minimizeToTray == true
    ) {
      // Aborting dialogs (if exists) which was initialized by "signal: abortController.signal"
      // Info: (for Linux) if we are hiding window when active messageBox is active - showing window back will lead to freezing it (it stay unresponsive)
      abortControllerAbort();
      // Prevent closing the window to be able to show it back immediately.
      // Just hide it.
      win.hide();
      event.preventDefault();
      return;

      // 'window-all-closed' event will be raised
      //return; // close window
    }

    event.preventDefault();
    if ((await isCanQuit()) == true) {
      isAppReadyToQuit = true;
      // application 'before-quit' event will be raised
      app.quit();
      return;
    }
  });

  win.on("closed", () => {
    win = null;
  });
}

async function applyMinimizedState() {
  let w = win;
  if (w == null) return null;
  const animate = false;
  if (store.state.settings.minimizedUI)
    return await w.setBounds({ width: config.MinimizedUIWidth }, animate);
  else return await w.setBounds({ width: config.MaximizedUIWidth }, animate);
}

function onDaemonExiting() {
  isAppReadyToQuit = true;
  app.quit();
}

// SETTINGS WINDOW
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

    frame: IsWindowHasFrame(),
  };

  settingsWindow = createBrowserWindow(windowConfig);

  if (process.env.WEBPACK_DEV_SERVER_URL) {
    // Load the url of the dev server if in development mode
    settingsWindow.loadURL(
      process.env.WEBPACK_DEV_SERVER_URL + `#settings/${viewName}`
    );
  } else {
    createProtocol("app");
    // Load the index.html when not in development
    settingsWindow.loadURL("app://./index.html" + `#settings/${viewName}`);
  }

  settingsWindow.once("ready-to-show", () => {
    settingsWindow.show();

    if (isDevelopment) {
      try {
        settingsWindow.webContents.openDevTools();
      } catch (e) {
        console.error("Failed to open dev tools:", e.toString());
      }
    }
  });
  settingsWindow.on("closed", () => {
    settingsWindow = null;
  });
}

function closeSettingsWindow() {
  if (settingsWindow == null) return;
  settingsWindow.destroy(); // close();
}
// UPDATE WINDOW
function createUpdateWindow() {
  if (updateWindow != null) {
    closeUpdateWindow();
  }

  let windowConfig = {
    backgroundColor: getBackgroundColor(),
    show: false,

    width: config.UpdateWindowWidth,
    height: 400,
    maxWidth: config.UpdateWindowWidth,
    maxHeight: 600,

    resizable: false,
    fullscreenable: false,
    maximizable: false,
    minimizable: false,

    center: true,
    title: "IVPN Update",

    autoHideMenuBar: true,

    frame: IsWindowHasFrame(),
  };

  updateWindow = createBrowserWindow(windowConfig);

  if (process.env.WEBPACK_DEV_SERVER_URL) {
    // Load the url of the dev server if in development mode
    updateWindow.loadURL(process.env.WEBPACK_DEV_SERVER_URL + `#update`);
  } else {
    createProtocol("app");
    // Load the index.html when not in development
    updateWindow.loadURL("app://./index.html" + `#update`);
  }

  updateWindow.once("ready-to-show", () => {
    updateWindow.show();

    if (isDevelopment) {
      try {
        updateWindow.webContents.openDevTools();
      } catch (e) {
        console.error("Failed to open dev tools:", e.toString());
      }
    }
  });

  updateWindow.on("closed", () => {
    updateWindow = null;
  });
}

function closeUpdateWindow() {
  if (updateWindow == null) return;
  updateWindow.destroy(); // close();
}

// INITIALIZE CONNECTION TO A DAEMON
async function connectToDaemon(
  doNotTryToInstall,
  isCanRetry,
  doNotTryToMacosStart
) {
  // MACOS ONLY: install daemon (privileged helper) if required
  if (Platform() === PlatformEnum.macOS && doNotTryToInstall !== true) {
    darwinDaemonInstaller.InstallDaemonIfRequired(
      () => {
        console.log("Installing daemon...");
        store.commit("daemonIsInstalling", true);
      }, //onInstallationStarted,
      (exitCode) => {
        // check if we still need to install helper
        darwinDaemonInstaller.IsDaemonInstallationRequired((code) => {
          if (code == 0) {
            // error: the helper not installed (we still detecting that helper must be installed (code == 0))
            console.error(
              `Error installing helper [code1: ${exitCode}, code2: ${code}]`
            );

            // set daemon state 'NotConnected'
            store.commit(
              "daemonConnectionState",
              DaemonConnectionType.NotConnected
            );

            // do not forget to notify that daemon installation is finished
            store.commit("daemonIsInstalling", false);
            // Skip connection to daemon
            return;
          }

          // daemon installation not required. Connecting to daemon...

          // force UI to show 'connecting' state
          store.commit(
            "daemonConnectionState",
            DaemonConnectionType.Connecting
          );

          // show/activate application window
          // (it can happen that app window is overlapped by another windows on a current moment)
          if (store.state.settings.minimizeToTray != true) menuOnShow();

          // wait some time to give Daemon chance to fully start
          setTimeout(async () => {
            // do not forget to notify that daemon installation is finished
            store.commit("daemonIsInstalling", false);

            // if success - try to connect to daemon with possibility to retry (wait until daemon start)
            // (doNotTryToInstall=true, isCanRetry=true)
            if (exitCode == 0)
              await connectToDaemon(true, true, doNotTryToMacosStart);
            else await connectToDaemon(true, false, doNotTryToMacosStart);
          }, 500);
        });
      } //onInstallationFinished
    );
    return;
  }

  let setConnState = function (state) {
    setTimeout(() => store.commit("daemonConnectionState", state), 0);
  };

  let onSetConnState = function (state) {
    // do not set 'NotConnected' state if we still trying to reconnect
    if (
      state === DaemonConnectionType.NotConnected &&
      store.state.daemonConnectionState !== DaemonConnectionType.Connected
    )
      return;

    store.commit("daemonConnectionState", state);
  };

  setConnState(DaemonConnectionType.Connecting);
  let connect = async function (retryNo) {
    try {
      await daemonClient.ConnectToDaemon(onSetConnState, onDaemonExiting);

      // initialize app updater
      StartUpdateChecker(OnAppUpdateAvailable);

      setConnState(DaemonConnectionType.Connected);
    } catch (e) {
      // MACOS ONLY: try to start daemon (privileged helper)
      if (Platform() === PlatformEnum.macOS && doNotTryToMacosStart != true) {
        darwinDaemonInstaller.TryStartDaemon();
        // wait some time to give Daemon chance to fully start
        setTimeout(async () => {
          // if success - try to connect to daemon with possibility to retry (wait until daemon start)
          // (doNotTryToInstall=true, isCanRetry=true, doNotTryToMacosStart=true)
          await connectToDaemon(true, true, true);
        }, 500);
        return;
      }

      if (
        e.unsupportedDaemonVersion === true ||
        isCanRetry != true ||
        retryNo > 15
      ) {
        setConnState(DaemonConnectionType.NotConnected);
      } else {
        // force UI to show 'connecting' state
        setConnState(DaemonConnectionType.Connecting);
        console.log(`Connecting to IVPN Daemon (retry #${retryNo}) ...`);
        setTimeout(async () => {
          await connect(retryNo + 1);
        }, 1000);
      }
    }
  };
  connect(1);
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
        params: { view: settingsViewName },
      };

      // Temporary navigate to '\'. This is required only if we already showing 'settings' view
      // (to be able to re-init 'settings' view with new parameters)
      win.webContents.send("main-change-view-request", "/");
      win.webContents.send("main-change-view-request", lastRouteArgs);
    }
  } catch (e) {
    console.log(e);
  }
}

// show\hide app from SYSTEM DOCK
function updateAppDockVisibility() {
  setAppDockVisibility(store.state.settings.showAppInSystemDock);
}

function setAppDockVisibility(isShow) {
  if (isShow) {
    // macOS
    if (app != null && app.dock != null) app.dock.show();

    // Windows
    if (win != null) {
      win.setSkipTaskbar(false);
    }
  } else {
    // macOS
    if (app != null && app.dock != null) {
      app.dock.hide(); // remove from dock
      if (win && win.isVisible()) win.show(); // ensure window is still shown (sometimes on macOS window is jumping under other window)
    }

    // Windows
    if (win != null) {
      win.setSkipTaskbar(true); // 'skip-taskbar' not applicable for Linux (since Electron v20)
    }
  }
}

// MENU ITEMS
function menuOnShow() {
  try {
    if (!win) {
      createWindow();
      win.show();
      win.focus();
    } else {
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

function menuOnCheckUpdates() {
  CheckUpdates();
  createUpdateWindow();
}

// UPDATE
function OnAppUpdateAvailable() {
  if (updateWindow) return;
  createUpdateWindow();
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
