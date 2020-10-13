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
        !state.accountStatus ||
        !state.session ||
        state.accountStatus.SessionTokenLastPart !==
          getLastPartOfSessionToken(state.session.Session)
      )
        state.accountStatus = null;
    },
    accountStatus(state, accState) {
      if (
        accState == null ||
        accState.Account == null ||
        (state.session &&
          state.session.Session &&
          accState.SessionToken !== state.session.Session)
      )
        return;
      state.accountStatus = accState.Account;

      // save session for account status object
      // (to be sure that account info belongs to correct session)
      if (accState.SessionToken)
        state.accountStatus.SessionTokenLastPart = getLastPartOfSessionToken(
          accState.SessionToken
        );

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

function getLastPartOfSessionToken(sessionToken) {
  if (!sessionToken || sessionToken.length < 6) return "";
  return sessionToken.substr(sessionToken.length - 6);
}
