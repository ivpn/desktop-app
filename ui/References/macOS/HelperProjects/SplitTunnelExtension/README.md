# IVPN Split Tunnel system extension

A macOS Network Extension of type *transparent proxy*
(`NETransparentProxyProvider`), shipped as a system extension inside
`IVPN.app`. While a session is running, macOS offers it every new outbound
TCP and UDP flow on the machine. Flows that belong to an excluded application
are relayed over the physical network interface; all other flows are declined
and keep following the system's routing, which with the VPN connected means
the tunnel.

Bundle identifier: `com.electron.ivpn-ui.SplitTunnel`. Requires macOS 12.

## Layout

| File | Contents |
|---|---|
| `main.m` | Process entry: logging setup, switch to the `ivpn-st` group, file-descriptor limit, `startSystemExtensionMode` |
| `STProxyProvider.m` / `.h` / `+Private.h` | Session lifecycle (`startProxyWithOptions:`, `stopProxyWithReason:`), the per-flow decision in `handleNewFlow:`, flow bookkeeping, UDP idle watchdog |
| `STProxyProvider+TCPRelay.m` | One relay connection per TCP flow, both pumps, connect deadline, teardown |
| `STProxyProvider+UDPRelay.m` | One relay connection per remote peer of a UDP flow, per-peer send cap, per-flow state |
| `STPathMatching.m` / `.h` | Turning a flow into an executable path and matching it against the excluded list, including process ancestry and bundle identifiers |
| `STPhysicalInterfaceSelector.m` / `.h` | Which physical interface relay connections are pinned to, kept live with path monitors |
| `STLog.m` / `.h` | Minimal leveled logger with a pluggable handler |
| `Info.plist`, `splittunnel.entitlements` | Bundle metadata and the extension's entitlements |
| `build.sh` | Builds and signs the `.systemextension` bundle |
| `Tests/` | Standalone tests, see below |

## How a flow is decided

`handleNewFlow:` is called for every new outbound flow on the machine, so it
does only cheap syscalls and no I/O:

1. Flows from IVPN's own bundle are never relayed (routing loop otherwise).
2. The flow's process is resolved to an executable path from its audit token.
   The path is compared with the excluded list: an `.app` bundle matches as a
   prefix, anything else must match exactly. Symlinked bundles are compared by
   their real path as well.
3. The flow's signing identifier is compared with the bundle identifiers of
   the excluded `.app` bundles, which also covers short-lived processes whose
   path can no longer be read.
4. If neither matched, the process ancestry is consulted: the kernel's
   responsible process first, then the parent chain up to launchd, both
   guarded against pid reuse. This is what excludes helpers, XPC services and
   commands started from an excluded terminal together with their parent.

Applications started through LaunchServices (`open`, Finder, login items) are
their own responsible process and are not inherited.

## Session start options

The host passes a dictionary to `startTunnelWithOptions:`; the provider reads:

| Key | Type | Meaning |
|---|---|---|
| `excludedPaths` | array of strings | Absolute `.app` bundle paths or executable paths to exclude. An empty array excludes nothing. |
| `debugLogging` | bool | Emit debug-level log lines for this session (off by default, see Logging). |
| `physicalInterface` | string | Optional. Pin relays to this interface, given as a BSD name or one of its IP addresses. |
| `physicalInterfaceType` | string | Optional. Pin relays to an interface type instead. |

Without the two optional keys the interface owning the `default` route is
used, re-evaluated on every network change. The IPv4 routing table is
consulted first and the IPv6 table only when IPv4 has no physical default.
Settings cannot be changed on a running session; the host stops and restarts
the session for every change, and `stopProxyWithReason:` closes every flow
that is being relayed.

## Runtime requirements

- The process switches to the `ivpn-st` group at startup. The IVPN firewall
  lets traffic of that group leave over the physical interface while every
  other way around the tunnel stays blocked. The group is created by the IVPN
  daemon; without it relayed traffic is blocked whenever the firewall is on.
- Each relayed TCP flow and each UDP peer costs one socket, so the soft
  file-descriptor limit is raised at startup.
- Entitlements: network extension (app-proxy provider) and the
  `group.com.electron.ivpn-ui` app group. The extension must not carry any
  hardened runtime relaxation entitlement. The host app that activates it has
  its own set, see `ui/References/macOS/build_HostAppEntitlements.plist`.

## Building

```sh
./build.sh -v <version> [-E <embedded.provisionprofile>]
ARCH_TARGET=x86_64 ./build.sh -v <version> -E ...   # cross-build; default is the host arch
```

The output is `bin/<arch>/com.electron.ivpn-ui.SplitTunnel.systemextension`.
The outer `ui/References/macOS/build.sh` calls this script, copies the bundle
into `IVPN.app/Contents/Library/SystemExtensions/` and re-signs it with
`splittunnel.entitlements` after the app's deep signing pass.

Activation on a Mac with System Integrity Protection enabled requires a
Developer ID signature and the embedded provisioning profile; both profiles
(host and extension) are required by the release build. macOS ignores a
rebuilt extension whose version equals the installed one, so bump the version
between test installs.

## Logging

Messages go to the unified log under the bundle identifier as subsystem:

```sh
/usr/bin/log stream --predicate 'subsystem == "com.electron.ivpn-ui.SplitTunnel"' --level debug
```

Info and error levels are always emitted. Debug level, which includes a line
for every new flow on the machine with its executable path, is emitted only
for a session started with `debugLogging`. The IVPN app sets that option when
it was launched with the `st-debug-logging` argument. To watch a test
session:

```sh
open -a IVPN --args st-debug-logging
/usr/bin/log stream --predicate 'subsystem == "com.electron.ivpn-ui.SplitTunnel"' --level debug
```

Start the stream before the traffic you want to see: the system keeps
debug-level lines in memory only, so `log show` will generally not have them
afterwards. In `zsh`, `log` is a shell builtin; use `/usr/bin/log`.

## Tests

```sh
./Tests/run_tests.sh
```

Compiles and runs the standalone tests with the command line tools only, no
Xcode project: path matching (bundle prefix, exact executable, symlink
resolution, process ancestry against real child processes) and the routing
table scan behind the default-route detection, fed hand-built tables for both
address families.

## Removing the extension during development

`systemextensionsctl uninstall` and `systemextensionsctl reset` need System
Integrity Protection disabled. With it enabled, deactivate through the app,
which is what the uninstaller does:

```sh
open -n -W -a /Applications/IVPN.app --args st-deactivate-and-quit
```

Alternatively delete the app bundle and reboot, or switch the extension off
in System Settings under Login Items & Extensions, Network Extensions.
