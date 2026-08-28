# IVPN Helper (`net.ivpn.client.Helper`)

A privileged helper tool installed via the macOS SMJobBless mechanism. Runs as root under `launchd`.

## Purpose

Fixes app bundle ownership on first run after a drag-install, then launches the IVPN Agent. On macOS, files copied from a DMG are owned by the installing user; the daemon requires root ownership to function correctly.

## How it works

On each invocation:
- If the bundle is already fully root-owned: launches the Agent directly.
- If not: verifies the Agent binary signature against the expected team identifier, fixes bundle ownership, then launches the Agent.

## Build

```sh
./build.sh -v <version> -c <APPLE_TEAM_ID>
```

`TEAM_IDENTIFIER` must be passed at compile time and must match the certificate used to sign the Agent binary.
