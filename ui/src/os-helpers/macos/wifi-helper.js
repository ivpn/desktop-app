//
//  UI for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2024 IVPN Limited.
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
  InitWifiHelper,
  InstallAgent,
  UninstallAgent,
};

import { Platform, PlatformEnum } from "@/platform/platform";
import daemonClient from "@/daemon-client";
import store from "@/store";

const { app, dialog } = require('electron');

const AGENT_STATUS = Object.freeze({
  NotRegistered: 0,
  Enabled: 1,
  RequiresApproval: 2,
  NotFound: 3,
});

const LOCATION_SERVICES_AUTHORISATION_STATUS = Object.freeze({
  NotDetermined: 0,
  Restricted: 1,
  Denied: 2,
  AuthorizedAlways: 3,
  AuthorizedWhenInUse: 4,
});

var isWarningDialogShown = false;

function InitWifiHelper(electronWindow, showSettingsFunc) {
  if (!isApplicable()) 
    return;
 
  // Check is addon available. 
  // In can be missing for old macOS versions (< v14.x)
  let helperAddon = null
  try {
    helperAddon = require('wifi-info-macos');
  } catch (e) {
      console.log("ERROR: (wifi-helper) wifi-info-macos addon not found");
      return;
  }

  // Just to ensure that required permission already requested
  helperAddon.LocationServicesRequestPermission();

  // update watrning message  
  helperAddon.LocationServicesSetAuthorizationChangeCallback(() => { setTimeout(onLocationServicesAuthorizationChange, 0) });
  updateWarningMessage();

  // Notify user about warnings (if they are)
  setTimeout(() => {showWarningDialogIfRequired(electronWindow, showSettingsFunc)}, 0);
  // Ensure that Agent in expected state (installed/unistalled)
  setTimeout(appyAgentState, 0);

  // subscribe to changes in a store
  store.subscribe((mutation) => {
    switch (mutation.type) {
      case "settings/daemonSettings":
        setTimeout(appyAgentState, 0);
        break;
      case "account/session":
        setTimeout(() => {showWarningDialogIfRequired(electronWindow, showSettingsFunc)}, 0);
        break;
      default:
    }
  });
}

function isApplicable() {
  if (Platform() !== PlatformEnum.macOS)
    return false;

  // Since macOS 14 Sonoma (Darwin v23.x.x) we have to use LaunchAgent to get WiFi info
  try {
    const os = require('os');
    const release = os.release();
    const versionParts = release.split('.').map(part => parseInt(part, 10));
    const majorVersion = versionParts[0];
    if (majorVersion < 23)
      return false; // Old macOS versions do not require LaunchAgent    
  } catch (e) {
    console.log("ERROR: (wifi-helper) Can not obtain macOS version:", e);    
  }
  return true
}

function isWifiFunctionalityEnabled() {
  let wifiSettings = store?.state?.settings?.daemonSettings?.WiFi;
  if (!wifiSettings) {
    console.log("ERROR: (wifi-helper) WiFi settings not found");
    return false;
  }
  return wifiSettings.trustedNetworksControl || wifiSettings.connectVPNOnInsecureNetwork;
}

async function showWarningDialogIfRequired(electronWindow, showSettingsFunc) {  
  try {
    if (isWarningDialogShown)
      return;
    if (!store.getters["account/isLoggedIn"])
      return;

    isWarningDialogShown = true;

    if (!electronWindow || !showSettingsFunc) 
      return;

    if (!isWifiFunctionalityEnabled())
      return;

    let errMsg = getOsConfigErrorDesctiption()
    if (errMsg) {
      let ret = await dialog.showMessageBox(electronWindow,
      {
        type: "warning",
        message: "WIFI Control is inactive",
        detail:  errMsg,
        buttons: ["OK", "System Settings ...", "Settings ..."],
      });  

      if (ret.response == 2) // WIFI Control settings
        showSettingsFunc();
      else if (ret.response == 1) // System Settings
      {
        const { shell } = require("electron");
        await shell.openExternal('x-apple.systempreferences:com.apple.preference.security?Privacy_LocationServices');
      }
    }  
  } catch (e) {
    console.error("ERROR: (wifi-helper) showWarningDialogIfRequired:", e);
  }
}

function appyAgentState() {
  try {    
    let helperAddon = require('wifi-info-macos');

    // INSTALL/UNINSTALL AGENT
    let agentStatus = helperAddon.AgentGetStatus();
    let isAgentRequired = isWifiFunctionalityEnabled();
 
    if (isAgentRequired && agentStatus != AGENT_STATUS.Enabled)
      InstallAgent()
    else if (!isAgentRequired && agentStatus == AGENT_STATUS.Enabled) {
      UninstallAgent()
      // request the daemon to refresh the current network info
      daemonClient.RequestWiFiCurrentNetwork();
    }

    updateWarningMessage();
  } catch (e) {
    console.error("ERROR: (wifi-helper) appyAgentState:", e);
  }
}

function InstallAgent() {
  if (!isApplicable()) 
    return;
  try {
    let helperAddon = require('wifi-info-macos');
    console.log("INFO: (wifi-helper) Installing agent");
    let ret = helperAddon.AgentInstall();
    if (ret != 0)
      console.error("ERROR: (wifi-helper) Failed to install agent:", ret);

  } catch (e) {
    console.error("ERROR: (wifi-helper) InstallAgent:", e);
  }
}

function UninstallAgent() {
  if (!isApplicable()) 
    return;
  try {
    let helperAddon = require('wifi-info-macos');    
    console.log("INFO: (wifi-helper) Uninstalling agent");
    let ret = helperAddon.AgentUninstall();
    if (ret != 0)
      console.error("ERROR: (wifi-helper) Failed to uninstall agent:", ret);
  } catch (e) {
    console.error("ERROR: (wifi-helper) UninstallAgent:", e);
  }
}

function onLocationServicesAuthorizationChange() {
  updateWarningMessage();
  // request the daemon to refresh the current network info
  daemonClient.RequestWiFiCurrentNetwork();
}

function updateWarningMessage() {
  let msg = getOsConfigErrorDesctiption();
  if (!msg) {
    if (isWifiFunctionalityEnabled()) {
      let helperAddon = require('wifi-info-macos');
      let status = helperAddon.AgentGetStatus()
      if (helperAddon.AgentGetStatus() != AGENT_STATUS.Enabled) {                
        msg = `Error: The IVPN LaunchAgent is not installed or not enabled (status: ${status}).`;
        console.log("ERROR: (wifi-helper):", msg);
      }
    }
  }

  store.commit("uiState/wifiWarningMessage", msg);
}

// The OS configuration may be in the state to Deny the app to use location services.
// This method returns the description of the error for user to understand what to do.
function getOsConfigErrorDesctiption() {
  let helperAddon = require('wifi-info-macos');

  let lsEnabled = helperAddon.LocationServicesEnabled();
  if (!lsEnabled) {
    // NOTE! The "Location Services," prefix is in use by components/settings/settings-networks.vue
    return "Location Services, essential for this functionality, are currently turned off in your System Settings.";
  }

  let lsAuthStatus = helperAddon.LocationServicesAuthorizationStatus();

  if (lsAuthStatus == LOCATION_SERVICES_AUTHORISATION_STATUS.NotDetermined) {
    // retry: sometimes it returns 'NotDetermined' on first call
    lsAuthStatus = helperAddon.LocationServicesAuthorizationStatus();
  }
  if (lsAuthStatus !== LOCATION_SERVICES_AUTHORISATION_STATUS.AuthorizedAlways 
    && lsAuthStatus !== LOCATION_SERVICES_AUTHORISATION_STATUS.AuthorizedWhenInUse) {      
      // NOTE! The "Location Services," prefix is in use by components/settings/settings-networks.vue 
      return `Location Services, essential for this functionality, are not currently activated for the ${app.getName()} application in your System Settings.`;
    }

  return "";
}