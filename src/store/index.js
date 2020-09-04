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

import Vue from "vue";
import Vuex from "vuex";
import { createSharedMutations } from "vuex-electron";

import { isStrNullOrEmpty } from "../helpers/helpers";

import account from "./module-account";
import vpnState from "./module-vpn-state";
import uiState from "./module-ui-state";
import settings from "./module-settings";

Vue.use(Vuex);

export default new Vuex.Store({
  plugins: [createSharedMutations()],

  modules: { account, vpnState, uiState, settings },
  strict: true,
  state: {
    isDaemonConnected: false,

    disabledFunctions: {
      WireGuardError: "",
      OpenVPNError: "",
      ObfsproxyError: ""
    },

    // Current location
    location: null,
    // true when we are requesting geo-lookup info on current moment
    isRequestingLocation: false
  },

  getters: {
    isWireGuardEnabled: state =>
      isStrNullOrEmpty(state.disabledFunctions.WireGuardError),
    isOpenVPNEnabled: state =>
      isStrNullOrEmpty(state.disabledFunctions.OpenVPNError),
    isObfsproxyEnabled: state =>
      isStrNullOrEmpty(state.disabledFunctions.ObfsproxyError)
  },

  // can be called from main process
  mutations: {
    replaceState(state, val) {
      Object.assign(state, val);
    },
    isDaemonConnected(state, isConnected) {
      state.isDaemonConnected = isConnected;
    },
    disabledFunctions(state, disabledFuncs) {
      state.disabledFunctions = disabledFuncs;
    },
    location(state, geoLocation) {
      state.location = geoLocation;
    },
    isRequestingLocation(state, value) {
      state.isRequestingLocation = value;
    }
  },

  // can be called from renderer
  actions: {}
});
