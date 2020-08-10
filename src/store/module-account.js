import { isStrNullOrEmpty } from "../helpers/helpers";

export default {
  namespaced: true,

  // STATE
  state: {
    // Session info
    session: {
      AccountID: "",
      Session: "",
      WgPublicKey: "",
      WgLocalIP: "",
      WgKeyGenerated: new Date(),
      WgKeysRegenIntervalSec: 0
    },
    accountStatus: {
      Active: false,
      ActiveUntil: 0,
      CurrentPlan: "",
      PaymentMethod: "",
      IsRenewable: false,
      WillAutoRebill: false,
      IsFreeTrial: false,
      Capabilities: [],
      Upgradable: false,
      UpgradeToPlan: "",
      UpgradeToURL: "",
      Limit: 0 // applicable for 'session limit' error
    }
  },

  mutations: {
    session(state, sessionInfo) {
      state.session = sessionInfo;

      // erase account state
      if (
        state.accountStatus == null ||
        state.session == null ||
        state.accountStatus.SessionToken !== state.session.Session
      )
        state.accountStatus = null;
    },
    accountStatus(state, accState) {
      if (
        state.session == null ||
        accState == null ||
        accState.Account == null ||
        accState.SessionToken == null ||
        accState.SessionToken !== state.session.Session
      )
        return;
      state.accountStatus = accState.Account;

      // save session for account status object
      // (to be sure that account info belongs to correct session)
      state.accountStatus.SessionToken = accState.SessionToken;

      // convert capabilities to lower case
      if (state.accountStatus.Capabilities != null)
        state.accountStatus.Capabilities.map(a => {
          return a.toLowerCase();
        });
    }
  },

  getters: {
    isLoggedIn: state => !isStrNullOrEmpty(state.session.Session),

    isAccountStateExists: state => {
      return state.accountStatus != null;
    },

    isMultihopAllowed: state => {
      return !(
        state.accountStatus == null ||
        state.accountStatus.Capabilities == null ||
        !state.accountStatus.Capabilities.includes("multihop")
      );
    }
  },

  actions: {
    accountStatus(context, val) {
      context.commit("accountStatus", val);

      if (context.getters.isMultihopAllowed === false)
        context.dispatch("settings/isMultiHop", false, { root: true });
    }
  },

  modules: {}
};
