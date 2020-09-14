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

import store from "@/store";
import daemonClient from "./daemon-client";

export function InitTrustedNetworks() {
  store.subscribe(mutation => {
    try {
      if (
        mutation.type === "vpnState/currentWiFiInfo" ||
        mutation.type === "settings/wifi"
      ) {
        setTimeout(async () => processWifiChange(), 0);
      }
    } catch (e) {
      console.error("Error in InitTrustedNetworks handler", e);
    }
  });
}

async function processWifiChange() {
  // 1. trying to apply rules (if network is configured)
  // 2. if network not configured -> apply rules for default trust status
  // 3. if default trust status not defined -> for insecure network: connect VPN

  // trusted networks config
  let wifi = store.state.settings.wifi;
  let networks = null;
  let actions = null;
  let trustedNetworksControl = false;
  if (wifi != null) {
    networks = wifi.networks;
    actions = wifi.actions;
    trustedNetworksControl = wifi.trustedNetworksControl;
  }

  // current network
  let currentWiFiInfo = store.state.vpnState.currentWiFiInfo;
  let isInsecureNetwork = null;
  let currSSID = null;
  if (currentWiFiInfo != null) {
    isInsecureNetwork = currentWiFiInfo.IsInsecureNetwork;
    currSSID = currentWiFiInfo.SSID;
  }

  let isProcessed = false;
  // if trusted network control is enabled
  if (trustedNetworksControl == true) {
    // try to apply configured network
    isProcessed = await applyTrustRuleForConfiguredNetwork(
      currSSID,
      networks,
      actions
    );

    if (!isProcessed) {
      // if network not configured - apply default trust operation for not configured networks
      let defaultTrustStatusTrusted = wifi.defaultTrustStatusTrusted;
      if (defaultTrustStatusTrusted != null) {
        await applyTrustRule(defaultTrustStatusTrusted, actions);
        isProcessed = true;
      }
    }
  }

  // insecure network (if network still not processed by 'trusted networks' functionality)
  if (!isProcessed) {
    if (
      isInsecureNetwork == true &&
      wifi.connectVPNOnInsecureNetwork == true &&
      !(
        store.getters["vpnState/isConnected"] ||
        store.getters["vpnState/isConnecting"]
      )
    ) {
      console.log(
        "Joined insecure network. Connecting (according to preferences) ..."
      );
      await daemonClient.Connect();
    }
  }
}

async function applyTrustRuleForConfiguredNetwork(currSSID, networks, actions) {
  if (!currSSID || networks == null || actions == null) return false;

  // check configuration for current network
  let networkConfigArr = networks.filter(wifi => wifi.ssid == currSSID);
  if (networkConfigArr == null || networkConfigArr.length == 0) return false;
  let networkConfig = networkConfigArr[0];

  if (networkConfig.isTrusted == null) return false;

  // apply trust operations for configured network
  await applyTrustRule(networkConfig.isTrusted, actions);

  return true;
}

async function applyTrustRule(isTrusted, actions) {
  if (isTrusted == null || actions == null) return;
  if (isTrusted) {
    // trusted
    if (
      actions.trustedDisconnectVpn == true &&
      !store.getters["vpnState/isDisconnected"]
    ) {
      console.log(
        "Joined trusted network. Disconnecting (according to preferences) ..."
      );
      await daemonClient.Disconnect();
    }
    if (
      actions.trustedDisableFirewall == true &&
      store.state.vpnState.firewallState.IsEnabled != false
    ) {
      console.log(
        "Joined trusted network. Disabling firewall (according to preferences) ..."
      );
      await daemonClient.EnableFirewall(false);
    }
  } else {
    // untrusted
    if (
      actions.unTrustedEnableFirewall == true &&
      store.state.vpnState.firewallState.IsEnabled != true
    ) {
      console.log(
        "Joined untrusted network. Enabling firewall (according to preferences) ..."
      );
      await daemonClient.EnableFirewall(true);
    }
    if (
      actions.unTrustedConnectVpn == true &&
      !(
        store.getters["vpnState/isConnected"] ||
        store.getters["vpnState/isConnecting"]
      )
    ) {
      console.log(
        "Joined untrusted network. Connecting (according to preferences) ..."
      );

      await daemonClient.Connect();
    }
  }
}
