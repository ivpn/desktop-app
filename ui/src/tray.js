/* eslint-disable no-undef */
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

const { Menu, Tray, app, nativeImage, dialog } = require("electron");
import store from "@/store";
import { PauseStateEnum } from "@/store/types";

import { VpnStateEnum } from "@/store/types";
import daemonClient from "@/daemon-client";
import { Platform, PlatformEnum } from "@/platform/platform";
import path from "path";

const { nativeTheme } = require("electron");

let tray = null;
let menuHandlerShow = null;
let menuHandlerPreferences = null;
let menuHandlerAccount = null;
let menuHandlerCheckUpdates = null;

let iconConnected = null;
let iconDisconnected = null;
let iconPaused = null;
let iconsConnecting = [];
let iconConnectingIdx = 0;
let iconConnectingIdxChanged = new Date().getTime();

let iconConnected_ForLightTheme = null; // (Windows-only) Black icon for light background
let iconDisconnected_ForLightTheme = null; // (Windows-only) Black icon for light background
let iconPaused_ForLightTheme = null; // (Windows-only) Black icon for light background
let iconsConnecting_ForLightTheme = []; // (Windows-only) Black icon for light background

// (Windows) We have to know information about the system theme (not the application theme)
// This is needed for determining the required color for the tray icons (white on a dark or black on a light)
// There is no possibility to detect the change of system theme! Therefore we are relying ONLY on the system theme at the application start!
// This property have to be initialized before any change of 'nativeTheme'
let useIconsForDarkTheme = true; // default - true: white icons
// Saving system theme before any changes of nativeTheme
try {
  if (Platform() == PlatformEnum.Windows) {
    useIconsForDarkTheme = nativeTheme.shouldUseDarkColors;
  }
} catch (e) {
  console.error(e);
}

export function InitTray(
  menuItemShow,
  menuItemPreferences,
  menuItemAccount,
  menuItemCheckUpdates
) {
  menuHandlerShow = menuItemShow;
  menuHandlerPreferences = menuItemPreferences;
  menuHandlerAccount = menuItemAccount;
  menuHandlerCheckUpdates = menuItemCheckUpdates;

  // load icons
  switch (Platform()) {
    case PlatformEnum.Windows:
      {
        nativeTheme.on("updated", () => {
          // the only way to detect real OS theme is when 'nativeTheme.themeSource == "system"'
          if (nativeTheme.themeSource == "system") {
            if (useIconsForDarkTheme != nativeTheme.shouldUseDarkColors) {
              useIconsForDarkTheme = nativeTheme.shouldUseDarkColors;
              updateTrayIcon();
            }
          }
        });

        let f = path.join(path.dirname(__dirname), "tray", "windows", "/");
        if (process.env.IS_DEBUG) {
          f = path.join(
            path.dirname(__dirname),
            "public",
            "tray",
            "windows",
            "/"
          );
        }

        iconConnected = nativeImage.createFromPath(f + "connected.ico");
        iconDisconnected = nativeImage.createFromPath(f + "disconnected.ico");
        iconPaused = nativeImage.createFromPath(f + "paused.ico");
        iconsConnecting.push(nativeImage.createFromPath(f + "connecting.ico"));
        // lightTheme
        iconConnected_ForLightTheme = nativeImage.createFromPath(
          f + "connected_lt.ico"
        );
        iconDisconnected_ForLightTheme = nativeImage.createFromPath(
          f + "disconnected_lt.ico"
        );
        iconPaused_ForLightTheme = nativeImage.createFromPath(
          f + "paused_lt.ico"
        );
        iconsConnecting_ForLightTheme.push(
          nativeImage.createFromPath(f + "connecting_lt.ico")
        );
      }
      break;
    case PlatformEnum.Linux:
      {
        const f = __static + "/tray/linux/";
        iconConnected = nativeImage.createFromPath(f + "connected.png");
        iconDisconnected = nativeImage.createFromPath(f + "disconnected.png");
        iconPaused = nativeImage.createFromPath(f + "paused.png");
        iconsConnecting.push(nativeImage.createFromPath(f + "connecting.png"));
      }
      break;
    case PlatformEnum.macOS:
      {
        const f = __static + "/tray/mac/";
        iconPaused = nativeImage.createFromPath(f + "pausedTemplate.png");
        iconConnected = nativeImage.createFromPath(f + "connectedTemplate.png");
        iconDisconnected = nativeImage.createFromPath(
          f + "disconnectedTemplate.png"
        );
        const c1 = nativeImage.createFromPath(f + "icon-1Template.png");
        const c2 = nativeImage.createFromPath(f + "icon-2Template.png");
        const c3 = nativeImage.createFromPath(f + "icon-3Template.png");
        const c4 = nativeImage.createFromPath(f + "icon-4Template.png");
        iconsConnecting.push(iconDisconnected, c1, c2, c3, c4, c3, c2, c1);
      }
      break;
  }

  // subscribe to any changes in a tore
  store.subscribe((mutation) => {
    try {
      switch (mutation.type) {
        case "uiState/isParanoidModePasswordView":
        case "vpnState/connectionState":
        case "vpnState/connectionInfo":
        case "vpnState/disconnected":
        case "vpnState/pauseState": {
          updateTrayMenu();
          updateTrayIcon();
          break;
        }
        case "settings/serverEntry":
        case "settings/serverExit":
        case "settings/isMultiHop":
        case "settings/isFastestServer":
        case "settings/isRandomServer":
        case "settings/isRandomExitServer":
        case "settings/serversFavoriteList":
        case "account/session":
          updateTrayMenu();
          break;

        case "isRequestingLocation":
        case "isRequestingLocationIPv6":
          updateTrayMenu();
          break;

        default:
      }
    } catch (e) {
      console.error("Error in store subscriber:", e);
    }
  });

  updateTrayMenu();
  updateTrayIcon();
}

function updateTrayIcon() {
  if (tray == null) return;
  if (store.getters["vpnState/isConnecting"]) {
    let icons = iconsConnecting;
    if (useIconsForDarkTheme === false) icons = iconsConnecting_ForLightTheme;

    tray.setImage(icons[iconConnectingIdx % icons.length]);
    if (icons.length > 1) {
      setTimeout(() => {
        let now = new Date().getTime();
        if (now - iconConnectingIdxChanged >= 200) {
          iconConnectingIdx += 1;
          iconConnectingIdxChanged = now;
        }
        updateTrayIcon();
      }, 200);
    }
    return;
  }

  iconConnectingIdx = 0;

  if (
    store.state.vpnState.pauseState === PauseStateEnum.Paused &&
    iconPaused != null
  )
    tray.setImage(
      useIconsForDarkTheme === false ? iconPaused_ForLightTheme : iconPaused
    );
  else if (store.state.vpnState.connectionState === VpnStateEnum.CONNECTED) {
    tray.setImage(
      useIconsForDarkTheme === false
        ? iconConnected_ForLightTheme
        : iconConnected
    );
  } else {
    tray.setImage(
      useIconsForDarkTheme === false
        ? iconDisconnected_ForLightTheme
        : iconDisconnected
    );
  }
}

function updateTrayMenu() {
  if (tray == null) {
    // eslint-disable-next-line no-undef
    tray = new Tray(iconDisconnected);

    tray.on("double-click", () => {
      if (menuHandlerShow != null) menuHandlerShow();
    });
  }

  // FAVORITE SERVERS MENU
  let favoriteSvrsTemplate = [];
  const favSvrs = store.state.settings.serversFavoriteList;
  if (favSvrs == null || favSvrs.length == 0) {
    favoriteSvrsTemplate = [
      { label: "No servers in favorite list", enabled: false },
    ];
  } else {
    favoriteSvrsTemplate = [{ label: "Connect to ...", enabled: false }];

    const serversHashed = store.state.vpnState.serversHashed;
    favSvrs.forEach((gw) => {
      const s = serversHashed[gw];
      if (s == null) return;

      var options = {};

      if (store.state.settings.isMultiHop) {
        options = {
          label: serverName(null, s),
          click: () => {
            menuItemConnect(null, s);
          },
        };
      } else {
        options = {
          label: serverName(s),
          click: () => {
            menuItemConnect(s);
          },
        };
      }

      favoriteSvrsTemplate.push(options);
    });
  }
  const favorites = Menu.buildFromTemplate(favoriteSvrsTemplate);

  // MAIN MENU

  var connectToName = "";
  if (
    (!store.state.settings.isMultiHop &&
      store.state.settings.isFastestServer) ||
    store.state.settings.isRandomServer
  )
    connectToName = serverName();
  else connectToName = serverName(store.state.settings.serverEntry);

  const isLoggedIn = store.getters["account/isLoggedIn"];

  // MAIN MENU
  var mainMenu = [];

  const statusText = GetStatusText();
  if (statusText) {
    const lines = statusText.split("\n");
    lines.forEach((l) => {
      if (l == "separator") mainMenu.push({ type: "separator" });
      else if (l) mainMenu.push({ label: l, enabled: false });
    });
    mainMenu.push({ type: "separator" });
  }

  mainMenu.push({ label: "Show IVPN", click: menuHandlerShow });
  mainMenu.push({ type: "separator" });

  const isPMPasswordView = store.state.uiState.isParanoidModePasswordView;
  if (isLoggedIn && isPMPasswordView !== true) {
    if (store.state.vpnState.connectionState === VpnStateEnum.DISCONNECTED) {
      mainMenu.push({
        label: `Connect to ${connectToName}`,
        click: () => menuItemConnect(),
      });
    } else {
      // PAUSE\RESUME
      if (store.state.vpnState.connectionState === VpnStateEnum.CONNECTED) {
        if (store.state.vpnState.pauseState === PauseStateEnum.Paused) {
          mainMenu.push({ label: "Resume", click: menuItemResume });
          const pauseSubMenuTemplate = [
            {
              label: "Resume in 5 min",
              click: () => {
                menuItemPause(5 * 60);
              },
            },
            {
              label: "Resume in 30 min",
              click: () => {
                menuItemPause(30 * 60);
              },
            },
            {
              label: "Resume in 1 hour",
              click: () => {
                menuItemPause(1 * 60 * 60);
              },
            },
            {
              label: "Resume in 3 hours",
              click: () => {
                menuItemPause(3 * 60 * 60);
              },
            },
          ];
          mainMenu.push({
            label: "Resume in",
            type: "submenu",
            submenu: Menu.buildFromTemplate(pauseSubMenuTemplate),
          });
          mainMenu.push({ type: "separator" });
        } else if (store.state.vpnState.pauseState === PauseStateEnum.Resumed) {
          const pauseSubMenuTemplate = [
            {
              label: "Pause for 5 min",
              click: () => {
                menuItemPause(5 * 60);
              },
            },
            {
              label: "Pause for 30 min",
              click: () => {
                menuItemPause(30 * 60);
              },
            },
            {
              label: "Pause for 1 hour",
              click: () => {
                menuItemPause(1 * 60 * 60);
              },
            },
            {
              label: "Pause for 3 hours",
              click: () => {
                menuItemPause(3 * 60 * 60);
              },
            },
          ];
          mainMenu.push({
            label: "Pause",
            type: "submenu",
            submenu: Menu.buildFromTemplate(pauseSubMenuTemplate),
          });
        }
      }
      mainMenu.push({ label: `Disconnect`, click: menuItemDisconnect });
    }

    mainMenu.push({
      label: "Favorite servers",
      type: "submenu",
      submenu: favorites,
    });
    mainMenu.push({ type: "separator" });
    mainMenu.push({ label: "Account", click: menuHandlerAccount });
    mainMenu.push({ label: "Settings", click: menuHandlerPreferences });
    if (menuHandlerCheckUpdates != null) {
      mainMenu.push({
        label: `Check for Updates`,
        click: menuHandlerCheckUpdates,
      });
    }
    mainMenu.push({ type: "separator" });
  }

  mainMenu.push({ label: "Quit", click: menuItemQuit });

  const contextMenu = Menu.buildFromTemplate(mainMenu);
  tray.setToolTip("IVPN Client");
  tray.setContextMenu(contextMenu);
}

function GetStatusText() {
  let retStr = "";

  if (store.getters["vpnState/isConnected"]) {
    retStr += `Connected: ${serverName()}`;
    if (store.state.vpnState.pauseState === PauseStateEnum.Paused) {
      retStr += ` (connection Paused)`;
    }
  } else if (store.getters["vpnState/isConnecting"]) retStr += "Connecting...";
  else if (store.getters["vpnState/isDisconnecting"])
    retStr += "Disconnecting...";
  else retStr += "Disconnected";

  // IPv4 location info
  let location = "";
  let l = store.state.location;
  if (l && !store.state.isRequestingLocation) {
    if (l.city) location += `${l.city}`;
    if (l.country) {
      if (location) location += `, `;
      location += `${l.country}`;
    }

    if (l.isIvpnServer == true) location += ` (ISP: IVPN)`;
    else if (l.isp) location += ` (ISP: ${l.isp})`;

    if (l.ip_address) retStr += `\nPublic IP: ${l.ip_address}`;
    if (location) retStr += `\nLocation: ${location}`;
  }

  // IPv6 location info
  let locationV6 = "";
  let l6 = store.state.locationIPv6;
  if (l6 && !store.state.isRequestingLocationIPv6 && l6.ip_address) {
    if (!store.getters["isIPv4andIPv6LocationsEqual"]) {
      if (l6.city) locationV6 += `${l6.city}`;
      if (l6.country) {
        if (locationV6) locationV6 += `, `;
        locationV6 += `${l6.country}`;
      }

      if (l6.isIvpnServer == true) locationV6 += ` (ISP: IVPN)`;
      else if (l6.isp) locationV6 += ` (ISP: ${l6.isp})`;
    }

    retStr += `\nseparator`;
    retStr += `\nPublic IPv6: ${l6.ip_address}`;
    if (locationV6) retStr += `\nLocation IPv6: ${locationV6}`;
  }

  return retStr;
}

function serverName(entryServer, exitServer) {
  function text(svr) {
    if (svr == null) return "";
    if (typeof svr === "string") return svr;
    return `${svr.city}, ${svr.country_code}`;
  }

  const isConnected = store.getters["vpnState/isConnected"];
  const isFastestServer = store.state.settings.isFastestServer;
  const isRandomServer = store.state.settings.isRandomServer;
  const isMultiHop = store.state.settings.isMultiHop;
  const isRandomExitServer = store.state.settings.isRandomExitServer;

  if (entryServer == null) {
    if (!isConnected && isFastestServer) entryServer = "Fastest Server";
    else if (!isConnected && isRandomServer) entryServer = "Random Server";
    else entryServer = store.state.settings.serverEntry;
  }

  if (exitServer == null && isMultiHop) {
    if (!isConnected && isRandomExitServer) exitServer = "Random Server";
    else exitServer = store.state.settings.serverExit;
  }

  var ret = text(entryServer);
  if (exitServer != null) ret = `${ret} -> ${text(exitServer)}`;
  return ret;
}

async function menuItemConnect(entrySvr, exitSvr) {
  try {
    await daemonClient.Connect(entrySvr, exitSvr);
  } catch (e) {
    console.error(e);
    dialog.showMessageBox({
      type: "error",
      buttons: ["OK"],
      message: `Failed to connect`,
      detail: e,
    });
  }
}

async function menuItemDisconnect() {
  try {
    await daemonClient.Disconnect();
  } catch (e) {
    console.error(e);
    dialog.showMessageBox({
      type: "error",
      buttons: ["OK"],
      message: `Failed to disconnect`,
      detail: e,
    });
  }
}

async function menuItemPause(PauseConnection) {
  try {
    await daemonClient.PauseConnection(PauseConnection);
  } catch (e) {
    console.error(e);
    dialog.showMessageBox({
      type: "error",
      buttons: ["OK"],
      message: `Failed to pause`,
      detail: e,
    });
  }
}

async function menuItemResume() {
  try {
    await daemonClient.ResumeConnection();
  } catch (e) {
    console.error(e);
    dialog.showMessageBox({
      type: "error",
      buttons: ["OK"],
      message: `Failed to resume`,
      detail: e,
    });
  }
}

function menuItemQuit() {
  app.quit();
}
