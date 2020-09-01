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

const { Menu, Tray, app, nativeImage } = require("electron");
import store from "@/store";
import { PauseStateEnum } from "@/store/types";

import { VpnStateEnum } from "@/store/types";
import daemonClient from "@/daemon-client";
import { Platform, PlatformEnum } from "@/platform/platform";

let tray = null;
let menuHandlerShow = null;
let menuHandlerPreferences = null;
let menuHandlerAccount = null;

let iconConnected = null;
let iconDisconnected = null;
let iconPaused = null;
let iconsConnecting = [];
let iconConnectingIdx = 0;
let iconConnectingIdxChanged = new Date().getTime();

export function InitTray(menuItemShow, menuItemPreferences, menuItemAccount) {
  menuHandlerShow = menuItemShow;
  menuHandlerPreferences = menuItemPreferences;
  menuHandlerAccount = menuItemAccount;

  // load icons
  switch (Platform()) {
    case PlatformEnum.Windows:
      iconConnected = nativeImage.createFromPath(
        // eslint-disable-next-line no-undef
        __static + "/tray/windows/connected.ico"
      );
      iconDisconnected = nativeImage.createFromPath(
        // eslint-disable-next-line no-undef
        __static + "/tray/windows/disconnected.ico"
      );
      iconsConnecting.push(
        nativeImage.createFromPath(
          // eslint-disable-next-line no-undef
          __static + "/tray/windows/connecting.ico"
        )
      );
      break;
    case PlatformEnum.Linux:
      iconConnected = nativeImage.createFromPath(
        // eslint-disable-next-line no-undef
        __static + "/tray/linux/connected.png"
      );
      iconDisconnected = nativeImage.createFromPath(
        // eslint-disable-next-line no-undef
        __static + "/tray/linux/disconnected.png"
      );
      iconsConnecting.push(
        nativeImage.createFromPath(
          // eslint-disable-next-line no-undef
          __static + "/tray/linux/connecting.png"
        )
      );
      break;
    case PlatformEnum.macOS:
      iconConnected = nativeImage.createFromPath(
        // eslint-disable-next-line no-undef
        __static + "/tray/mac/icon-connectedTemplate.png"
      );
      iconDisconnected = nativeImage.createFromPath(
        // eslint-disable-next-line no-undef
        __static + "/tray/mac/icon-disconnectedTemplate.png"
      );
      iconPaused = nativeImage.createFromPath(
        // eslint-disable-next-line no-undef
        __static + "/tray/mac/icon-pausedTemplate.png"
      );
      iconsConnecting.push(
        nativeImage.createFromPath(
          // eslint-disable-next-line no-undef
          __static + "/tray/mac/icon-connectingTemplate.png"
        )
      );
      break;
  }

  // subscribe to any changes in a tore
  store.subscribe(mutation => {
    try {
      switch (mutation.type) {
        case "vpnState/connectionState":
        case "vpnState/connectionInfo":
        case "vpnState/disconnected":
        case "vpnState/pauseState": {
          updateTrayMenu();
          updateTrayIcon();
          break;
        }
        case "settings/serverEntry":
        case "settings/isMultiHop":
        case "settings/isFastestServer":
        case "settings/isRandomServer":
        case "settings/serversFavoriteList":
        case "account/session":
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
    tray.setImage(iconsConnecting[iconConnectingIdx % iconsConnecting.length]);
    if (iconsConnecting.length > 1) {
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
    tray.setImage(iconPaused);
  else if (store.state.vpnState.connectionState === VpnStateEnum.CONNECTED) {
    tray.setImage(iconConnected);
  } else {
    tray.setImage(iconDisconnected);
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
      { label: "No servers in favorite list", enabled: false }
    ];
  } else {
    favoriteSvrsTemplate = [{ label: "Connect to ...", enabled: false }];

    const serversHashed = store.state.vpnState.serversHashed;
    favSvrs.forEach(gw => {
      const s = serversHashed[gw];
      if (s == null) return;

      var options = {
        label: serverName(
          s,
          store.state.settings.isMultiHop
            ? store.state.settings.serverExit
            : null
        ),
        click: () => {
          menuItemConnect(s);
        }
      };
      favoriteSvrsTemplate.push(options);
    });
  }
  const favorites = Menu.buildFromTemplate(favoriteSvrsTemplate);

  // MAIN MENU
  var connectToName = "";
  if (store.state.settings.isFastestServer) connectToName = "Fastest Server";
  else if (store.state.settings.isRandomServer) connectToName = "Random Server";
  else
    connectToName = serverName(
      store.state.settings.serverEntry,
      store.state.settings.isMultiHop ? store.state.settings.serverExit : null
    );

  const isLoggedIn = store.getters["account/isLoggedIn"];

  var mainMenu = [
    { label: `IVPN v${app.getVersion()}`, enabled: false },
    { type: "separator" },
    { label: "Show IVPN", click: menuHandlerShow },
    { type: "separator" }
  ];
  if (isLoggedIn) {
    if (store.state.vpnState.connectionState === VpnStateEnum.DISCONNECTED) {
      mainMenu.push({
        label: `Connect to ${connectToName}`,
        click: () => menuItemConnect()
      });
    } else {
      // PAUSE\RESUME
      if (
        process.platform !== "linux" &&
        store.state.vpnState.connectionState === VpnStateEnum.CONNECTED
      ) {
        if (store.state.vpnState.pauseState === PauseStateEnum.Paused)
          mainMenu.push({ label: "Resume", click: menuItemResume });
        else if (store.state.vpnState.pauseState === PauseStateEnum.Resumed) {
          const pauseSubMenuTemplate = [
            {
              label: "Pause for 5 min",
              click: () => {
                menuItemPause(5 * 60);
              }
            },
            {
              label: "Pause for 30 min",
              click: () => {
                menuItemPause(30 * 60);
              }
            },
            {
              label: "Pause for 1 hour",
              click: () => {
                menuItemPause(1 * 60 * 60);
              }
            },
            {
              label: "Pause for 3 hours",
              click: () => {
                menuItemPause(3 * 60 * 60);
              }
            }
          ];
          mainMenu.push({
            label: "Pause",
            type: "submenu",
            submenu: Menu.buildFromTemplate(pauseSubMenuTemplate)
          });
        }
      }
      mainMenu.push({ label: `Disconnect`, click: menuItemDisconnect });
    }

    mainMenu.push({
      label: "Favorite servers",
      type: "submenu",
      submenu: favorites
    });
    mainMenu.push({ type: "separator" });
    mainMenu.push({ label: "Account", click: menuHandlerAccount });
    mainMenu.push({ label: "Settings", click: menuHandlerPreferences });
    mainMenu.push({ type: "separator" });
  }
  mainMenu.push({ label: "Quit", click: menuItemQuit });

  const contextMenu = Menu.buildFromTemplate(mainMenu);
  tray.setToolTip("IVPN Client");
  tray.setContextMenu(contextMenu);
}

function serverName(server, exitSvr) {
  if (server == null) return "";
  var ret = `${server.city}, ${server.country_code}`;
  if (exitSvr != null)
    ret = `${ret} -> ${exitSvr.city}, ${exitSvr.country_code}`;
  return ret;
}

function menuItemConnect(entrySvr) {
  try {
    daemonClient.Connect(entrySvr);
  } catch (e) {
    console.error(e);
  }
}

function menuItemDisconnect() {
  try {
    daemonClient.Disconnect();
  } catch (e) {
    console.error(e);
  }
}

function menuItemPause(PauseConnection) {
  try {
    daemonClient.PauseConnection(PauseConnection);
  } catch (e) {
    console.error(e);
  }
}

function menuItemResume() {
  try {
    daemonClient.ResumeConnection();
  } catch (e) {
    console.error(e);
  }
}

function menuItemQuit() {
  app.quit();
}
