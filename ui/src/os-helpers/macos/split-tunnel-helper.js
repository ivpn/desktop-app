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
  StopAndWait,
  InstallExtension,
  UninstallExtension,
  UninstallExtensionAndWait,
  RecheckExtensionStateOnFocus,
};

import { Platform, PlatformEnum } from "@/platform/platform";
import {
  SplitTunnelMacExtStateEnum,
  SplitTunnelMacSessionStatusEnum,
  DaemonConnectionType,
} from "@/store/types";
import store from "@/store";

// Tracks whether activation has already been requested this run, so a
// stream of status updates while ST stays enabled doesn't resubmit the
// request repeatedly - re-armed on the next launch (see Init below), which
// is what actually catches a bundled extension version upgrade.
let _extensionActivationRequested = false;

// Mirrors the extensionState from the last onStateChanged callback (or the
// initial getExtensionState() read) - used by RecheckExtensionStateOnFocus() below
// so it doesn't need to touch the store/getter from this main-process module.
let _lastExtensionState = SplitTunnelMacExtStateEnum.NotInstalled;

// Serialized options of the last config actually handed to the addon, or null
// when the session is stopped (see ApplyConfig/Stop).
let _lastAppliedConfig = null;

// Whether the proxy configuration was registered with the OS (or a session
// applied, which registers it too) since Split Tunnel was enabled. Once per
// enable cycle: re-registering after the user declined the system prompt
// would raise that prompt again on every status update.
let _configRegistrationRequested = false;

// Set by StopAndWait(); invoked from the addon.onStateChanged() handler in
// Init() once the session reports a stopped status.
let _onSessionStopped = null;

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
    if (_onSessionStopped && isSessionStopped(s.sessionStatus)) _onSessionStopped();
    // A session that dies while a config is applied usually means the user
    // switched the extension off in System Settings - ask the OS.
    if (_lastAppliedConfig && isSessionStopped(s.sessionStatus)) addon.refreshExtensionState();
    // Activation completing is not a daemon status change, so nothing else
    // re-triggers the config applied while the extension was still installing.
    if (!wasInstalled && s.extensionState === SplitTunnelMacExtStateEnum.Installed) {
      _lastAppliedConfig = null; // that config never reached a running extension
      applyDaemonStatus(store.state.vpnState.splitTunnelling);
    }
    // The extension was removed behind our back (e.g. `systemextensionsctl
    // uninstall`) after we activated it: activate it again, which brings the
    // approval prompt and its banner back instead of silently doing nothing.
    if (s.extensionState === SplitTunnelMacExtStateEnum.NotInstalled && _extensionActivationRequested) {
      _extensionActivationRequested = false;
      applyDaemonStatus(store.state.vpnState.splitTunnelling);
    }
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
  // VPN state changes have to re-trigger this too (see applyDaemonStatus):
  // the daemon's SplitTunnelStatus payload is identical whether the VPN is up
  // or down, so it alone never signals a transition.
  const stRetriggerMutations = [
    "daemonConnectionState",
    "vpnState/splitTunnelling",
    "vpnState/connectionState",
    "vpnState/connectionInfo",
    "vpnState/disconnected",
  ];
  store.subscribe((mutation) => {
    if (stRetriggerMutations.includes(mutation.type)) {
      applyDaemonStatus(store.state.vpnState.splitTunnelling);
    }
  });
  applyDaemonStatus(store.state.vpnState.splitTunnelling);

  // initialState above is the addon's own cache ("notInstalled" on a fresh
  // process); ask the OS for the real state - it arrives via onStateChanged.
  addon.refreshExtensionState();
}

// cfg is the resolved Split Tunnel state computed by the daemon, forwarded
// verbatim as the tunnel session's start options.
function ApplyConfig(cfg) {
  const addon = getAddon();
  if (!addon) return;

  // The daemon broadcasts SplitTunnelStatus on many unrelated events (VPN
  // connect, DNS change, login...) and applying a config always restarts the
  // session, which force-closes every relayed connection - so only act on a
  // config that actually differs.
  const serialized = JSON.stringify(cfg || {});
  if (serialized === _lastAppliedConfig) return;
  _lastAppliedConfig = serialized;

  // The extension may have been switched off or removed since it was last
  // seen; a failed start alone does not tell. Ask the OS alongside the apply.
  addon.refreshExtensionState();
  addon.applyConfig(cfg || {});
}

// Maps the daemon's SplitTunnelStatus shape onto the addon's start options.
function applyDaemonStatus(status) {
  // While the daemon is not connected the store holds defaults, not facts
  // (every connect attempt resets the VPN state to "disconnected"), so acting
  // on them would stop the session on every reconnect. The real state arrives
  // with the connection and re-triggers this via "daemonConnectionState".
  if (store.state.daemonConnectionState !== DaemonConnectionType.Connected) return;
  if (!status) return;
  if (!status.IsEnabled) {
    _configRegistrationRequested = false;
    Stop();
    return;
  }
  if (!_extensionActivationRequested) {
    _extensionActivationRequested = true;
    InstallExtension();
  }
  // Nothing to start (or register) until the extension is actually installed:
  // saving the proxy configuration raises a system prompt that makes no sense
  // while the extension itself is still waiting for the user's approval. The
  // "installed" transition in Init() re-runs this.
  if (_lastExtensionState !== SplitTunnelMacExtStateEnum.Installed) return;
  // Same two gates Windows applies (isDriverMustBeDisabled in
  // daemon/splittun/splittun_windows.go): a running session is offered every
  // flow on the machine, so it must have both a tunnel to bypass and
  // something to exclude.
  const isVpnActive =
    store.getters["vpnState/isConnected"] && !store.getters["vpnState/isPaused"];
  if (!isVpnActive || !status.SplitTunnelApps?.length) {
    Stop();
    // Raise the system's "add proxy configurations" prompt now, while the user
    // is still enabling Split Tunnel, rather than on the first VPN connect.
    if (!_configRegistrationRequested) {
      _configRegistrationRequested = true;
      RegisterConfig();
    }
    return;
  }
  _configRegistrationRequested = true; // applying a config registers it as well
  ApplyConfig({
    isInversed: status.IsInversed, // not yet consumed by the extension - reserved for a future inverse-mode implementation
    excludedPaths: status.SplitTunnelApps,
    // Extension debug logging follows the app's logging setting; a change
    // takes effect with the next session (re)start.
    debugLogging: false, //!!store.state.settings.daemonSettings?.IsLogging,
  });
}

function Stop() {
  const addon = getAddon();
  if (!addon) return;
  _lastAppliedConfig = null;
  addon.stop();
}

function isSessionStopped(sessionStatus) {
  return (
    sessionStatus === SplitTunnelMacSessionStatusEnum.Disconnected ||
    sessionStatus === SplitTunnelMacSessionStatusEnum.Invalid
  );
}

// Called by background.js on quit. The extension session outlives this
// process and nothing else controls it (the daemon cannot), so a closed UI
// must mean no running session - otherwise excluded apps would keep bypassing
// the VPN (and the kill switch) with no way to change or stop it.
// Returns false when there is nothing to stop; true when a stop was issued,
// in which case onDone() is invoked once the session reports stopped, or
// after a short timeout so quitting can never hang on the extension.
function StopAndWait(onDone) {
  const addon = getAddon();
  if (!addon || isSessionStopped(addon.getSessionStatus())) return false;
  _onSessionStopped = onDone;
  // Fallback: the stopped status may never arrive (e.g. the proxy manager
  // failed to load), and the addon's stop() is asynchronous - so quitting
  // without waiting at all could exit before the stop request was sent.
  // Calling onDone() twice is harmless (app.quit() is idempotent).
  setTimeout(onDone, 3000);
  Stop();
  return true;
}

function InstallExtension() {
  const addon = getAddon();
  if (!addon) return;
  addon.installExtension();
}

function RegisterConfig() {
  const addon = getAddon();
  if (!addon) return;
  addon.registerConfig();
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

// Called by background.js whenever an app window regains focus. The addon's
// state is a cache, so returning from System Settings after enabling,
// disabling or removing the extension needs an explicit re-query (approval
// itself is reported by the pending activation request; the addon skips the
// probe while one is pending). Only once activation has been requested, so a
// user who never enabled Split Tunnel is never probed.
function RecheckExtensionStateOnFocus() {
  if (!_extensionActivationRequested) return;
  const addon = getAddon();
  if (addon) addon.refreshExtensionState();
}
