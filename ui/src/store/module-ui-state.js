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

import { SplitTunnelMacExtStateEnum } from "./types";

export default {
  namespaced: true,

  state: {
    isParanoidModePasswordView: false,

    // favorite servers view selected
    serversFavoriteView: false,

    currentSettingsViewName: null, // 'account', 'general', 'version' ...

    isIPv6View: false,

    isPauseResumeInProgress: false,
    //{
    //  state: AppUpdateStage.Downloading,
    //  error: null,
    //  readyToInstallBinary: "",
    //  readyToInstallSignatureFile: "",
    //  downloadStatus: {
    //    contentLength: 0,
    //    downloaded:    0
    //  }
    //}
    appUpdateProgress: null,

    // if not empty, then UI settings view will show this message 
    // (e.g. message text about the Location Services permission required for ability to get WiFi info)
    wifiWarningMessage: "",

    // macOS only: last state reported by the Split Tunnel system extension/session
    // addon (ui/addons/split-tunnel-macos) - { extensionState, sessionStatus, lastError } - or null before it has reported anything
    splitTunnelMacOS: null,
  },

  getters: {
    // macOS only: why Split Tunnel is enabled but not working, or "" when it is
    // (or when it is disabled). Only states the user has to act on; normal
    // progress (installing) is not reported. Shared by the Settings banner and
    // the main view, so they never disagree.
    splitTunnelMacOSIssue(state, getters, rootState) {
      const extState = state.splitTunnelMacOS;
      if (!extState || !rootState.vpnState.splitTunnelling?.IsEnabled) return "";
      switch (extState.extensionState) {
        case SplitTunnelMacExtStateEnum.NeedsUserApproval:
          return "Split Tunnel is enabled but not active yet. Approve the IVPN system extension in System Settings; it starts automatically once approved.";
        case SplitTunnelMacExtStateEnum.Disabled:
          return "The Split Tunnel system extension is switched off. Enable it in System Settings to use Split Tunnel.";
        case SplitTunnelMacExtStateEnum.NeedsReboot:
          return "Restart your Mac to finish installing the Split Tunnel system extension.";
        case SplitTunnelMacExtStateEnum.Error:
          return `Split Tunnel system extension error: ${extState.lastError || "unknown error"}`;
        case SplitTunnelMacExtStateEnum.Installed:
          // e.g. the user did not allow adding the proxy configuration
          if (extState.lastError)
            return `Split Tunnel could not start: ${extState.lastError}. Disable and re-enable Split Tunnel to retry.`;
          return "";
        default:
          return "";
      }
    },
  },

  mutations: {
    isParanoidModePasswordView(state, value) {
      state.isParanoidModePasswordView = value;
    },
    serversFavoriteView(state, value) {
      state.serversFavoriteView = value;
    },
    appUpdateProgress(state, value) {
      state.appUpdateProgress = value;
    },
    currentSettingsViewName(state, value) {
      state.currentSettingsViewName = value;
    },
    isIPv6View(state, value) {
      state.isIPv6View = value;
    },
    isPauseResumeInProgress(state, value) {
      state.isPauseResumeInProgress = value;
    },
    wifiWarningMessage(state, value) {
      state.wifiWarningMessage = value;
    },
    splitTunnelMacOS(state, value) {
      state.splitTunnelMacOS = value;
    },
  },

  // can be called from renderer
  actions: {
    isParanoidModePasswordView(context, value) {
      context.commit("isParanoidModePasswordView", value);
    },
    serversFavoriteView(context, value) {
      context.commit("serversFavoriteView", value);
    },
    currentSettingsViewName(context, value) {
      context.commit("currentSettingsViewName", value);
    },
    isIPv6View(context, value) {
      context.commit("isIPv6View", value);
    },
  },
};
