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

import { app, protocol, BrowserWindow, Menu, dialog } from "electron";
import {
  createProtocol,
  installVueDevtools
} from "vue-cli-plugin-electron-builder/lib";

// start waiting for events from Renderer processes
import "./ipc/main-listener";

import store from "@/store";
import daemonClient from "./daemon-client";
import { InitPersistentSettings, SaveSettings } from "./settings-persistent";
import { InitConnectionResumer } from "./connection-resumer";
import { IsWindowHasTitle } from "@/platform/platform";

const isDevelopment = process.env.NODE_ENV !== "production";

InitPersistentSettings();
InitConnectionResumer();
connectToDaemon();

// Keep a global reference of the window object, if you don't, the window will
// be closed automatically when the JavaScript object is garbage collected.
let win;

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
            if (win !== null) win.webContents.send("change-view-request", "/");
          }
        } /*,
        {
          label: "UI macOS",
          click() {
            if (win !== null)
              win.webContents.send("change-ui-style", PlatformEnum.macOS);
          }
        },
        {
          label: "UI Linux",
          click() {
            if (win !== null)
              win.webContents.send("change-ui-style", PlatformEnum.Linux);
          }
        },
        {
          label: "UI Windows",
          click() {
            if (win !== null)
              win.webContents.send("change-ui-style", PlatformEnum.Windows);
          }
        }*/
      ]
    }
  ]);
  Menu.setApplicationMenu(dockMenu);
}

let titleBarStyle = "default";
if (!IsWindowHasTitle()) titleBarStyle = "hiddenInset"; //"hidden";

// CREATE WINDOW
function createWindow() {
  // Create the browser window.
  win = new BrowserWindow({
    width: 800,
    height: 600,
    minWidth: 700,
    minHeight: 550,
    maxWidth: 1600,
    maxHeight: 1200,

    center: true,
    title: "IVPN",

    fullscreenable: false,
    titleBarStyle: titleBarStyle,
    autoHideMenuBar: true,

    webPreferences: {
      enableRemoteModule: true,
      // Use pluginOptions.nodeIntegration, leave this alone
      // See nklayman.github.io/vue-cli-plugin-electron-builder/guide/security.html#node-integration for more info
      // nodeIntegration: process.env.ELECTRON_NODE_INTEGRATION
      nodeIntegration: true
    }
  });

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

  win.on("closed", () => {
    win = null;
  });
}

// INITIALIZE CONNECTION TO A DAEMON
function connectToDaemon() {
  try {
    daemonClient.ConnectToDaemon();
  } catch (e) {
    console.error(e);
  }
}

// Quit when all windows are closed.
app.on("window-all-closed", () => {
  // if we are waiting to save settings - save it immediately
  SaveSettings();

  // On macOS it is common for applications and their menu bar
  // to stay active until the user quits explicitly with Cmd + Q
  if (process.platform !== "darwin") {
    app.quit();
  }
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
  // request geolocation
  // (asynchronously, to do not block application start)
  setTimeout(() => {
    const api = require("./api");
    try {
      // the successful result will be saved in store
      api.default.GeoLookup();
    } catch (e) {
      console.error(`Failed to determine geolocation: ${e}`);
    }
  }, 0);

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
    if (store.state.settings.firewallOffOnExit) {
      await daemonClient.EnableFirewall(false);
    }
    return;
  }

  let actionNo = 0;
  if (store.state.settings.quitWithoutConfirmation) {
    actionNo = store.state.settings.disconnectOnQuit ? 0 : 1;
  } else {
    actionNo = dialog.showMessageBoxSync({
      type: "question",
      message: "Are you sure want to quit?",
      detail: "You are connected to the VPN.",
      buttons: ["Disconnect VPN & Quit", "Keep VPN connected & Quit", "Cancel"]
    });
  }
  switch (actionNo) {
    case 2: // Cancel
      event.preventDefault();
      break;

    case 1: // Exit & Keep VPN connection
      // just close application
      break;

    case 0: // Exit & Disconnect VPN
      try {
        if (
          store.state.settings.firewallOnOffOnConnect ||
          store.state.settings.disconnectOnQuit
        )
          daemonClient.EnableFirewall(false);

        daemonClient.Disconnect();
      } catch (e) {
        console.log(e);
      }
      break;
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

// subscribe to any changes in a tore
store.subscribe(mutation => {
  try {
    if (mutation.type === "settings/showAppInSystemDock") {
      updateAppDockVisibility();
    }
  } catch (e) {
    console.error("Error in store subscriber:", e);
  }
});

// show\hide app from system dock
function updateAppDockVisibility() {
  if (store.state.settings.showAppInSystemDock){ 
    // macOS
    if (app != null && app.dock != null)
      app.dock.show();

    // Windows
    if (win != null) {
      win.setSkipTaskbar(false)
    }
  }
  else {
    // macOS
    if (app != null && app.dock != null)
      app.dock.hide(); // remove from dock

    // Windows
    if (win != null) {
      win.setSkipTaskbar(true);
    }

    // ensure window is still visible
    if (win != null) win.show(); 
  }
}
