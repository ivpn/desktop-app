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

const electron = window.require("electron");
const { ipcRenderer } = electron;

async function invoke(channel, ...args) {
  try {
    return await ipcRenderer.invoke(channel, ...args);
  } catch (e) {
    console.error(e);
    // remove prefix error text
    // (like: Error occurred in handler for 'renderer-request-login': Error: ...)
    const regexp = new RegExp(`^.+ '${channel}': (Error:)*`);
    const errStr = `${e}`;
    const corrected = errStr.replace(regexp, "");
    if (errStr === corrected) throw e;
    throw new Error(corrected);
  }
}

export default {
  ConnectToDaemon: async () => {
    return await invoke("renderer-request-connect-to-daemon");
  },
  RefreshStorage: () => {
    // function using to re-apply all mutations
    // This is required to send to renderer processes current storage state
    return invoke("renderer-request-refresh-storage");
  },

  Login: async (accountID, force) => {
    return await invoke("renderer-request-login", accountID, force);
  },
  Logout: async () => {
    return await invoke("renderer-request-logout");
  },
  AccountStatus: async () => {
    return await invoke("renderer-request-account-status");
  },
  PingServers: () => {
    return invoke("renderer-request-ping-servers");
  },
  Connect: async (entryServer, exitServer) => {
    // if entryServer or exitServer is null -> will be used current selected servers
    // otherwise -> current selected servers will be replaced by a new values before connect
    return await invoke("renderer-request-connect", entryServer, exitServer);
  },
  Disconnect: async () => {
    return await invoke("renderer-request-disconnect");
  },

  PauseConnection: async pauseSeconds => {
    if (pauseSeconds == null) return;
    //var pauseTill = new Date();
    //pauseTill.setSeconds(pauseTill.getSeconds() + pauseSeconds);
    return await invoke("renderer-request-pause-connection", pauseSeconds);
  },
  ResumeConnection: async () => {
    return await invoke("renderer-request-resume-connection");
  },

  EnableFirewall: async isEnable => {
    return await invoke("renderer-request-firewall", isEnable);
  },
  KillSwitchSetAllowLANMulticast: async isEnable => {
    return await invoke(
      "renderer-request-KillSwitchSetAllowLANMulticast",
      isEnable
    );
  },
  KillSwitchSetAllowLAN: async isEnable => {
    return await invoke("renderer-request-KillSwitchSetAllowLAN", isEnable);
  },
  KillSwitchSetIsPersistent: async isEnable => {
    return await invoke("renderer-request-KillSwitchSetIsPersistent", isEnable);
  },

  SetLogging: async () => {
    return await invoke("renderer-request-set-logging");
  },
  SetObfsproxy: async () => {
    return await invoke("renderer-request-set-obfsproxy");
  },

  SetDNS: async antitrackerIsEnabled => {
    return await invoke("renderer-request-set-dns", antitrackerIsEnabled);
  },
  GeoLookup: async () => {
    return await invoke("renderer-request-geolookup");
  },

  WgRegenerateKeys: async () => {
    return await invoke("renderer-request-wg-regenerate-keys");
  },
  WgSetKeysRotationInterval: async intervalSec => {
    return await invoke(
      "renderer-request-wg-set-keys-rotation-interval",
      intervalSec
    );
  },

  GetWiFiAvailableNetworks: async () => {
    return await invoke("renderer-request-wifi-get-available-networks");
  }
};
