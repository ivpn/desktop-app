# split-tunnel-macos

Node.js native addon (N-API, Objective-C) used by the IVPN UI on macOS to
control the Split Tunnel system extension and its proxy session.

## Why this exists

Two macOS frameworks are involved in Split Tunnel and both can only be used
from a process that runs in the user's login session and lives inside an app
bundle with the matching entitlements:

- `SystemExtensions.framework` (`OSSystemExtensionManager`) activates,
  deactivates and inspects the system extension.
- `NetworkExtension.framework` (`NETransparentProxyManager`,
  `NETunnelProviderSession`) registers the proxy configuration and starts or
  stops the proxy session.

The IVPN daemon is a root LaunchDaemon and cannot do either, so the Electron
main process does it through this addon. The daemon stays the single source
of truth for the Split Tunnel configuration; the UI only executes it and
reports the extension state back.

The consumer is `ui/src/os-helpers/macos/split-tunnel-helper.js`. Nothing
else should call the addon directly.

## Layout

| Path | Contents |
|---|---|
| `src/addon.m` | The whole implementation: an `STSplitTunnelController` singleton plus the N-API glue at the end of the file |
| `lib/binding.js` | The JavaScript API, a thin wrapper over the exported functions |
| `binding.gyp` | node-gyp build definition (ARC, deployment target 12.0) |

## Building

The addon is a `file:` dependency of `ui/package.json`, so `npm install` in
`ui/` builds it with node-gyp. To rebuild it alone:

```sh
cd ui/addons/split-tunnel-macos
npx node-gyp rebuild
```

The result is `build/Release/split-tunnel-macos-native.node`. The addon is
macOS only (`"os": ["darwin"]`); on other platforms the helper never loads it.

## API

All functions are synchronous calls into the addon; the work they start is
asynchronous and its result arrives through `onStateChanged`.

```js
const st = require("split-tunnel-macos");

st.onStateChanged(({ extensionState, sessionStatus, lastError }) => { ... });

st.getExtensionState();      // cached, see values below
st.refreshExtensionState();  // ask the OS; result arrives via onStateChanged
st.installExtension();       // submit an activation request
st.uninstallExtension();     // submit a deactivation request

st.getSessionStatus();       // cached NEVPNStatus as a string
st.registerConfig();         // save the proxy configuration without starting
st.applyConfig(cfg);         // start (or restart) the session with cfg as start options
st.stop();                   // stop the session
```

Extension states: `notInstalled`, `installing`, `needsUserApproval`,
`needsReboot`, `installed`, `disabled` (installed but switched off in System
Settings), `error`. Session statuses follow `NEVPNStatus`:
`invalid`, `disconnected`, `connecting`, `connected`, `reasserting`,
`disconnecting`.

`cfg` is passed to the extension unchanged as the session start options. The
keys the extension reads are documented in
`ui/References/macOS/HelperProjects/SplitTunnelExtension/README.md`. The
helper sets `debugLogging` when the app was launched with the
`st-debug-logging` argument (`open -a IVPN --args st-debug-logging`).

## Behaviour worth knowing

- **Extension identity.** The extension's bundle identifier is the host app's
  identifier with `.SplitTunnel` appended, and it is loaded from
  `Contents/Library/SystemExtensions` of the running app bundle.
- **Activation.** Only one activation request is in flight at a time. While
  it is pending, which can be as long as the user takes to approve it in
  System Settings, no properties probe is submitted: the OS would cancel the
  activation as superseded, and the activation's own completion is what
  reports the approval. A superseded error is not reported as an extension
  error.
- **Probes.** `refreshExtensionState` submits a properties request. Several
  may be in flight; a probe whose completion never arrives does not block
  later ones. A probe that finds the extension switched off in System
  Settings reports `disabled`; one that finds it removed reports
  `notInstalled`.
- **Applying a configuration** always stops the running session and starts a
  new one with the new options; the provider does not support live updates.
  The restart waits for the session to report disconnected, with a timeout,
  and a start that fails while the previous session is still tearing down is
  retried a few times. Commands are numbered so that of several apply or stop
  calls issued in quick succession only the last one takes effect.
- **Proxy configuration prompt.** The first `saveToPreferences` raises the
  system's "would like to add proxy configurations" prompt. `registerConfig`
  exists so the UI can raise it while the user is enabling Split Tunnel
  rather than on the first VPN connect. The text of that prompt cannot be
  customised.
- **Extension replacement.** When the OS replaces a running extension with a
  newer bundled version, the addon restarts an active session so the new
  binary takes over.

## Requirements on the host app

The host app must be signed with the `system-extension.install`,
`networkextension` (app-proxy provider) and app-group entitlements and carry
an embedded provisioning profile that authorises them. Among the hardened
runtime relaxations only `allow-jit` is permitted on a system extension
host; the others make macOS refuse to launch the app. See
`ui/References/macOS/build_HostAppEntitlements.plist`.

## Debugging

Request failures, configuration save failures and session start failures are
logged with `NSLog` under the `[split-tunnel-macos]` prefix and show up in
the unified log as messages of the `IVPN` process:

```sh
/usr/bin/log stream --predicate 'eventMessage CONTAINS "split-tunnel-macos"'
```

Requests to the system extension daemon can be followed with
`process == "sysextd"` in the same tool. Note that in `zsh`, `log` is a shell
builtin; use `/usr/bin/log`.
