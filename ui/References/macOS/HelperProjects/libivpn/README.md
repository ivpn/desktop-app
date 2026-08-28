# libivpn (OBSOLETE)

> **This project is no longer used and is kept for reference only.**

A native shared library (`libivpn.dylib`) from the original Mono (C#/.NET) UI era.

## Original purpose

- **XPC listener**: allowed the daemon to notify the Mono UI of its TCP port and connection secret via a privileged Mach XPC service.
- **Helper management**: SMJobBless wrappers for installing and removing the privileged helper.
- **Power change notifications**: system sleep/wake monitoring.

## Why it is obsolete

When the UI was rewritten from Mono to Electron, the XPC notification path was replaced by a simple file read. The `libivpn` build tag that activates this code is disabled in all production builds.
