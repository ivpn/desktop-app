# IVPN Installer / Uninstaller

A dual-purpose binary compiled twice with different `IS_INSTALLER` flags - once as the installer and once as the uninstaller.

## Purpose

Installs and removes the privileged helper tool (`net.ivpn.client.Helper`) using the macOS SMJobBless mechanism. Installing a privileged `LaunchDaemon` requires explicit user authorization, which this binary handles.

## How it works

- **Installer** (`IS_INSTALLER=1`): checks if the correct helper version is already installed. If not, prompts for authorization and calls `SMJobBless` to install or upgrade it.
- **Uninstaller** (`IS_INSTALLER=0`): prompts for authorization and removes the helper binary and its `LaunchDaemon` plist.

## Build

```sh
./build.sh -c <APPLE_TEAM_ID>
```
