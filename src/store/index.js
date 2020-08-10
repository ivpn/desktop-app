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
    location: null
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
    }
  },

  // can be called from renderer
  actions: {}
});
