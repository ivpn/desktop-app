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
