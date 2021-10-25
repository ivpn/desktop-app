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

export default {
  namespaced: true,

  state: {
    // favorite servers view selected
    serversFavoriteView: false,
    pauseConnectionTill: null, // Date()

    currentSettingsViewName: null, // 'account', 'general', 'version' ...

    isIPv6View: false,

    //{
    //  state: AppUpdaterStageEnum.Downloading,
    //  error: null,
    //  readyToInstallBinary: "",
    //  readyToInstallSignatureFile: "",
    //  downloadStatus: {
    //    contentLength: 0,
    //    downloaded:    0
    //  }
    //}
    appUpdateProgress: null
  },

  mutations: {
    serversFavoriteView(state, value) {
      state.serversFavoriteView = value;
    },
    pauseConnectionTill(state, value) {
      state.pauseConnectionTill = value;
    },
    appUpdateProgress(state, value) {
      state.appUpdateProgress = value;
    },
    currentSettingsViewName(state, value) {
      state.currentSettingsViewName = value;
    },
    isIPv6View(state, value) {
      state.isIPv6View = value;
    }
  },

  // can be called from renderer
  actions: {
    serversFavoriteView(context, value) {
      context.commit("serversFavoriteView", value);
    },
    pauseConnectionTill(context, value) {
      context.commit("pauseConnectionTill", value);
    },
    currentSettingsViewName(context, value) {
      context.commit("currentSettingsViewName", value);
    },
    isIPv6View(context, value) {
      context.commit("isIPv6View", value);
    }
  }
};
