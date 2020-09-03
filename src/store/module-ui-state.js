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

export default {
  namespaced: true,

  state: {
    // current view is default 'control view' (with button 'connect'... etc.; NOT servers view)
    isDefaultControlView: true,
    // favorite servers view selected
    serversFavoriteView: false,
    pauseConnectionTill: null //new Date()
  },

  mutations: {
    isDefaultControlView(state, value) {
      state.isDefaultControlView = value;
    },
    serversFavoriteView(state, value) {
      state.serversFavoriteView = value;
    },
    pauseConnectionTill(state, value) {
      state.pauseConnectionTill = value;
    }
  },

  // can be called from renderer
  actions: {
    isDefaultControlView(context, value) {
      context.commit("isDefaultControlView", value);
    },
    serversFavoriteView(context, value) {
      context.commit("serversFavoriteView", value);
    },
    pauseConnectionTill(context, value) {
      context.commit("pauseConnectionTill", value);
    }
  }
};
