//
//  UI for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2026 IVPN Limited.
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

const addon = require("../build/Release/split-tunnel-macos-native");

module.exports = {
  // "notInstalled" | "installing" | "needsUserApproval" | "needsReboot" | "installed" | "error"
  getExtensionState: () => addon.ExtensionGetState(),
  installExtension: () => addon.ExtensionActivate(),
  uninstallExtension: () => addon.ExtensionDeactivate(),
  // "invalid" | "disconnected" | "connecting" | "connected" | "reasserting" | "disconnecting"
  getSessionStatus: () => addon.SessionGetStatus(),
  // cfg is forwarded verbatim as NETunnelProviderSession start options; always a
  // full stop-then-restart of the session, never a live in-place update.
  applyConfig: (cfg) => addon.SessionApplyConfig(JSON.stringify(cfg || {})),
  stop: () => addon.SessionStop(),
  // cb receives { extensionState, sessionStatus, lastError }, called whenever
  // either the extension state or the tunnel session status changes.
  onStateChanged: (cb) => {
    addon.SetStateChangedCallback((json) => {
      try {
        cb(JSON.parse(json));
      } catch (e) {
        cb({ extensionState: "error", sessionStatus: "invalid", lastError: String(e) });
      }
    });
  },
};
