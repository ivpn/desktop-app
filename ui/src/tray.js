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
import { VpnStateEnum, ColorThemeTrayIcon } from "@/store/types";
import { CheckAndNotifyInaccessibleServer } from "@/helpers/helpers_servers";

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

const EnumMenuId = Object.freeze({
  CommonSeparator: "CommonSeparator",
  Connect: "Connect",
  Pause: "Pause",
  Resume: "Resume",
  ResumeIn: "ResumeIn",
  Disconnect: "Disconnect",
  Favorite: "Favorite",
});

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

        // lightTheme
        iconConnected_ForLightTheme = nativeImage.createFromPath(
          f + "connected_lt.png"
        );
        iconDisconnected_ForLightTheme = nativeImage.createFromPath(
          f + "disconnected_lt.png"
        );
        iconPaused_ForLightTheme = nativeImage.createFromPath(
          f + "paused_lt.png"
        );
        iconsConnecting_ForLightTheme.push(
          nativeImage.createFromPath(f + "connecting_lt.png")
        );
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
        case "vpnState/disconnected": {
          updateTrayMenu();
          updateTrayIcon();
          break;
        }
        case "settings/serverEntry":
        case "settings/serverExit":
        case "settings/serverEntryHostId":
        case "settings/serverExitHostId":
        case "settings/isMultiHop":
        case "settings/isFastestServer":
        case "settings/isRandomServer":
        case "settings/isRandomExitServer":
        case "settings/serversFavoriteList":
        case "settings/hostsFavoriteListDnsNames":
        case "settings/showHosts":
        case "account/session":
          updateTrayMenu();
          break;
        case "settings/colorThemeTrayIcon":
          updateTrayIcon();
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

  let isLightIcons = useIconsForDarkTheme !== false;

  switch (store.state.settings.colorThemeTrayIcon) {
    case ColorThemeTrayIcon.light:
      isLightIcons = true;
      break;
    case ColorThemeTrayIcon.dark:
      isLightIcons = false;
      break;
    default:
      break;
  }

  if (store.getters["vpnState/isConnecting"]) {
    let icons = iconsConnecting;
    if (isLightIcons === false) icons = iconsConnecting_ForLightTheme;
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

  if (store.getters["vpnState/isPaused"] && iconPaused != null)
    tray.setImage(
      isLightIcons === false ? iconPaused_ForLightTheme : iconPaused
    );
  else if (store.state.vpnState.connectionState === VpnStateEnum.CONNECTED) {
    tray.setImage(
      isLightIcons === false ? iconConnected_ForLightTheme : iconConnected
    );
  } else {
    tray.setImage(
      isLightIcons === false ? iconDisconnected_ForLightTheme : iconDisconnected
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

  // favorite servers/hosts for current protocol
  let favSvrs = store.getters["settings/favoriteServersAndHosts"];

  if (favSvrs == null || favSvrs.length == 0) {
    favoriteSvrsTemplate = [
      { label: "No servers in favorite list", enabled: false },
    ];
  } else {
    favoriteSvrsTemplate = [];

    favSvrs.forEach((s) => {
      let host = null;
      if (s.favHostParentServerObj && s.favHost) {
        host = s.favHost;
        s = s.favHostParentServerObj;
      }

      if (!s || !s.gateway) return;

      var options = null;
      if (store.state.settings.isMultiHop) {
        options = {
          label: serverName(null, s, null, host),
          click: async () => {
            menuItemConnect(null, s, null, host);
          },
        };
      } else {
        options = {
          label: serverName(s, null, host, null),
          click: () => {
            menuItemConnect(s, null, host, null);
          },
        };
      }

      if (options) favoriteSvrsTemplate.push(options);
    });

    // sort
    favoriteSvrsTemplate = favoriteSvrsTemplate.slice().sort((a, b) => {
      return a.label.localeCompare(b.label);
    });
    // add 'header'
    favoriteSvrsTemplate.unshift({ label: "Connect to ...", enabled: false });
  }

  const favorites = Menu.buildFromTemplate(favoriteSvrsTemplate);

  // MAIN MENU
  var connectToName = serverName();

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
        id: EnumMenuId.Connect,
      });
    } else {
      // PAUSE\RESUME
      if (store.state.vpnState.connectionState === VpnStateEnum.CONNECTED) {
        if (store.getters["vpnState/isPaused"]) {
          mainMenu.push({
            label: "Resume",
            click: menuItemResume,
            id: EnumMenuId.Resume,
          });
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
            id: EnumMenuId.ResumeIn,
            submenu: Menu.buildFromTemplate(pauseSubMenuTemplate),
          });
          mainMenu.push({ type: "separator", id: EnumMenuId.CommonSeparator });
        } else if (!store.getters["vpnState/isPaused"]) {
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
            id: EnumMenuId.Pause,
            submenu: Menu.buildFromTemplate(pauseSubMenuTemplate),
          });
        }
      }
      mainMenu.push({
        label: `Disconnect`,
        click: menuItemDisconnect,
        id: EnumMenuId.Disconnect,
      });
    }

    mainMenu.push({
      label: "Favorite servers",
      type: "submenu",
      submenu: favorites,
      id: EnumMenuId.Favorite,
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

  // APPLY TRAY MENU
  mainMenu.push({ label: "Quit", click: menuItemQuit });

  tray.setToolTip("IVPN Client");
  tray.setContextMenu(Menu.buildFromTemplate(mainMenu));

  updateDockMenuMacOS(mainMenu);
}

function updateDockMenuMacOS(mainMenuTemplate) {
  // macOS: APPLY DOCK MENU
  if (process.platform !== "darwin") return;
  try {
    // Minimize amount of calls: app.dock.setMenu(...);
    //
    // There is a scenario the Dock menu items handlers will not be triggered:
    //    User opened the dock menu and then the dock menu have been updated in the background
    //    In this situation  the already opened menu became not workable (clicking on menu items will have no effect)
    if (
      store.getters["vpnState/isConnecting"] ||
      store.getters["vpnState/isDisconnecting"]
    ) {
      // We are in connecting/disconnecting stage. Just erasing the dock menu for now (to avoid the situation described above).
      // The Dock menu will be updated soon (as soon as connected/disconnected)
      app.dock.setMenu(Menu.buildFromTemplate([]));
    } else {
      // only specific items can be shown in Dock menu
      let dockMenuSource = mainMenuTemplate.filter(
        (el) =>
          el.id == EnumMenuId.CommonSeparator ||
          el.id == EnumMenuId.Connect ||
          el.id == EnumMenuId.Disconnect ||
          el.id == EnumMenuId.Pause ||
          el.id == EnumMenuId.Resume ||
          el.id == EnumMenuId.ResumeIn ||
          el.id == EnumMenuId.Favorite
      );

      const oldMenu = app.dock.getMenu();
      const newMenu = Menu.buildFromTemplate(dockMenuSource);

      if (isMenuEquals(oldMenu, newMenu) !== true) app.dock.setMenu(newMenu);
    }
  } catch (e) {
    console.error(e);
  }
}

function isMenuEquals(menu1, menu2) {
  if (menu1 == menu2) return true;
  if ((!menu1 && menu2) || (menu1 && !menu2)) return false;
  if (menu1.items.length != menu2.items.length) return false;
  const len = menu1.items.length;
  for (let i = 0; i < len; i++) {
    const it1 = menu1.items[i];
    const it2 = menu2.items[i];
    if (it1.label != it2.label) return false;
    if (!isMenuEquals(it1.submenu, it2.submenu)) return false;
  }
  return true;
}

function GetStatusText() {
  let retStr = "";

  if (store.getters["vpnState/isConnected"]) {
    retStr += `Connected: ${serverName()}`;
    if (store.getters["vpnState/isPaused"]) retStr += ` (connection Paused)`;
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

function serverName(entryServer, exitServer, entryServerHost, exitServerHost) {
  function text(svr, svrHostId) {
    if (svr == null) return "";
    if (typeof svr === "string") return svr;

    let ret = `${svr.city}, ${svr.country_code}`;
    if (svrHostId) {
      ret += " (" + svrHostId.split(".")[0] + ")";
    }
    return ret;
  }

  const isConnected = store.getters["vpnState/isConnected"];
  const isFastestServer = store.state.settings.isFastestServer;
  const isRandomServer = store.state.settings.isRandomServer;
  const isMultiHop = store.state.settings.isMultiHop;
  const isRandomExitServer = store.state.settings.isRandomExitServer;

  let entryServerHostId = "";
  let exitServerHostId = "";
  if (entryServerHost && entryServerHost.hostname)
    entryServerHostId = entryServerHost.hostname;
  if (exitServerHost && exitServerHost.hostname)
    exitServerHostId = exitServerHost.hostname;

  if (entryServer == null) {
    if (!isConnected && isFastestServer && !isMultiHop) {
      entryServer = "Fastest Server";
      entryServerHostId = "";
    } else if (!isConnected && isRandomServer) {
      entryServer = "Random Server";
      entryServerHostId = "";
    } else {
      entryServer = store.state.settings.serverEntry;
      entryServerHostId = store.state.settings.serverEntryHostId;
    }
  }

  if (exitServer == null && isMultiHop) {
    if (!isConnected && isRandomExitServer) {
      exitServer = "Random Server";
      exitServerHostId = "";
    } else {
      exitServer = store.state.settings.serverExit;
      exitServerHostId = store.state.settings.serverExitHostId;
    }
  }

  var ret = text(entryServer, entryServerHostId);
  if (exitServer != null)
    ret = `${ret} -> ${text(exitServer, exitServerHostId)}`;
  return ret;
}

async function menuItemConnect(entrySvr, exitSvr, entryHost, exitHost) {
  try {
    if (
      exitSvr &&
      (await CheckAndNotifyInaccessibleServer(true, exitSvr)) !== true
    )
      return;

    if (entrySvr) {
      store.dispatch("settings/serverEntry", entrySvr);
      store.dispatch("settings/isFastestServer", false);
      store.dispatch("settings/isRandomServer", false);
    }

    if (exitSvr) {
      store.dispatch("settings/serverExit", exitSvr);
      store.dispatch("settings/isRandomExitServer", false);
    }

    if (entryHost && entryHost.hostname)
      store.dispatch("settings/serverEntryHostId", entryHost.hostname);
    if (exitHost && exitHost.hostname)
      store.dispatch("settings/serverExitHostId", exitHost.hostname);

    await daemonClient.Connect();
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
