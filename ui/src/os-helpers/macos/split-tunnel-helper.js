//
//  UI for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2026 IVPN Limited.
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

// Electron main-process wrapper around the 'split-tunnel-macos' native addon.
// The daemon (root LaunchDaemon) cannot drive OSSystemExtensionManager /
// NETransparentProxyManager itself - both require a user-session process
// inside an app bundle - so this is the only place that talks to them.
// The daemon stays the single source of truth for Split Tunnel configuration;
// this module only executes what it's told and reports state back.

export default {
  Init,
  ApplyConfig,
  Stop,
  InstallExtension,
  UninstallExtension,
  UninstallExtensionAndWait,
  RecheckApprovalOnFocus,
};

import { Platform, PlatformEnum } from "@/platform/platform";
import { SplitTunnelMacExtStateEnum } from "@/store/types";
import store from "@/store";

function isApplicable() {
  return Platform() === PlatformEnum.macOS;
}

function getAddon() {
  if (!isApplicable()) return null;
  try {
    return require("split-tunnel-macos");
  } catch (e) {
    console.error("ERROR: (split-tunnel-helper) split-tunnel-macos addon not found:", e);
    return null;
  }
}

// Call once, at app startup. Extension activation itself is deferred until
// Split Tunnel is actually enabled (see applyDaemonStatus) - a user who
// never enables it should never see the OS's "system extension blocked"
// approval prompt.
// onStateChangedCallback (optional) is invoked with { extensionState,
// sessionStatus, lastError } on every change - used by background.js to
// report extension readiness back to the daemon via daemonClient, without
// this module needing to depend on daemon-client itself.
function Init(onStateChangedCallback) {
  const addon = getAddon();
  if (!addon) return;

  addon.onStateChanged((s) => {
    const wasInstalled = _lastExtensionState === SplitTunnelMacExtStateEnum.Installed;
    _lastExtensionState = s.extensionState;
    store.commit("uiState/splitTunnelMacOS", s);
    if (onStateChangedCallback) onStateChangedCallback(s);
    // Activation completing is not a daemon status change, so nothing else
    // re-triggers the config applied while the extension was still installing.
    if (!wasInstalled && s.extensionState === SplitTunnelMacExtStateEnum.Installed)
      applyDaemonStatus(store.state.vpnState.splitTunnelling);
  });
  const initialState = {
    extensionState: addon.getExtensionState(),
    sessionStatus: addon.getSessionStatus(),
    lastError: "",
  };
  _lastExtensionState = initialState.extensionState;
  store.commit("uiState/splitTunnelMacOS", initialState);
  if (onStateChangedCallback) onStateChangedCallback(initialState);

  // Self-contained, like wifi-helper.js: watch the store for the daemon's
  // resolved Split Tunnel status ourselves, rather than daemon-client.js
  // reaching into this module - keeps daemon-client.js platform-agnostic.
  store.subscribe((mutation) => {
    if (mutation.type === "vpnState/splitTunnelling") {
      applyDaemonStatus(mutation.payload);
    }
  });
  applyDaemonStatus(store.state.vpnState.splitTunnelling);
}

// cfg is the resolved Split Tunnel state computed by the daemon, forwarded
// verbatim as the tunnel session's start options.
function ApplyConfig(cfg) {
  const addon = getAddon();
  if (!addon) return;
  addon.applyConfig(cfg || {});
}

// Tracks whether activation has already been requested this run, so a
// stream of status updates while ST stays enabled doesn't resubmit the
// request repeatedly - re-armed on the next launch (see Init above), which
// is what actually catches a bundled extension version upgrade.
let _extensionActivationRequested = false;

// Mirrors the extensionState from the last onStateChanged callback (or the
// initial getExtensionState() read) - used by RecheckApprovalOnFocus() below
// so it doesn't need to touch the store/getter from this main-process module.
let _lastExtensionState = SplitTunnelMacExtStateEnum.NotInstalled;

// Maps the daemon's SplitTunnelStatus shape onto the addon's start options.
function applyDaemonStatus(status) {
  if (!status) return;
  if (!status.IsEnabled) {
    Stop();
    return;
  }
  if (!_extensionActivationRequested) {
    _extensionActivationRequested = true;
    InstallExtension();
  }
  ApplyConfig({
    isInversed: status.IsInversed, // not yet consumed by the extension - reserved for a future inverse-mode implementation
    excludedPaths: status.SplitTunnelApps,
  });
}

function Stop() {
  const addon = getAddon();
  if (!addon) return;
  addon.stop();
}

function InstallExtension() {
  const addon = getAddon();
  if (!addon) return;
  addon.installExtension();
}

function UninstallExtension() {
  const addon = getAddon();
  if (!addon) return;
  addon.uninstallExtension();
}

// Same as UninstallExtension(), but invokes onDone() once the deactivation
// OSSystemExtensionRequest actually completes (success or failure) instead of
// leaving the caller to guess how long that takes - used by background.js's
// disposable 'st-deactivate-and-quit' process to know when it's safe to quit.
function UninstallExtensionAndWait(onDone) {
  const addon = getAddon();
  if (!addon) { onDone(); return; }
  addon.onStateChanged(() => onDone());
  addon.uninstallExtension();
}

// Called by background.js whenever the main window regains focus. macOS only
// notifies this process of an approval change via a fresh activation request
// (getExtensionState() just returns a cached value, it doesn't re-query OS
// approval state) - so re-submit one, but only while the user is actually
// mid-approval, to avoid spamming OSSystemExtensionRequest on every focus.
function RecheckApprovalOnFocus() {
  if (_lastExtensionState !== SplitTunnelMacExtStateEnum.NeedsUserApproval) return;
  InstallExtension();
}
