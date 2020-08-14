const { Menu, Tray, app } = require("electron");
import store from "@/store";
import { VpnStateEnum } from "@/store/types";
import daemonClient from "@/daemon-client";

let tray = null;
let menuHandlerShow = null;
let menuHandlerPreferences = null;
let menuHandlerAccount = null;
export function InitTray(menuItemShow, menuItemPreferences, menuItemAccount) {
  menuHandlerShow = menuItemShow;
  menuHandlerPreferences = menuItemPreferences;
  menuHandlerAccount = menuItemAccount;

  // subscribe to any changes in a tore
  store.subscribe(mutation => {
    try {
      switch (mutation.type) {
        case "settings/serverEntry":
        case "settings/isMultiHop":
        case "settings/isFastestServer":
        case "settings/isRandomServer":
        case "settings/serversFavoriteList":
        case "vpnState/connectionState":
        case "vpnState/disconnected":
        case "account/session":
          updateTrayMenu();
          break;
        default:
      }
    } catch (e) {
      console.error("Error in store subscriber:", e);
    }
  });
}
function updateTrayMenu() {
  if (tray == null) {
    // eslint-disable-next-line no-undef
    tray = new Tray(__static + "/icon-disconnected.png");
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
    { label: "Show IVPN", click: menuHandlerShow },
    { label: "About", click: menuItemAbout },
    { type: "separator" }
  ];
  if (isLoggedIn) {
    if (store.state.vpnState.connectionState === VpnStateEnum.DISCONNECTED) {
      mainMenu.push({
        label: `Connect to ${connectToName}`,
        click: () => menuItemConnect()
      });
    } else mainMenu.push({ label: `Disconnect`, click: menuItemDisconnect });
    mainMenu.push({
      label: "Favorite servers",
      type: "submenu",
      submenu: favorites
    });
    mainMenu.push({ type: "separator" });
    mainMenu.push({ label: "Account", click: menuHandlerAccount });
    mainMenu.push({ label: "Preferences", click: menuHandlerPreferences });
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

function menuItemAbout() {
  app.setAboutPanelOptions({
    copyright: null,
    website: "https://www.ivpn.net"
  });
  app.showAboutPanel();
}

function menuItemQuit() {
  app.quit();
}
