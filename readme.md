# IVPN Daemon for IVPN Client Desktop (Windows/macOS)

**IVPN Daemon for IVPN Client Desktop** is a core module of IVPN Client for Windows and macOS built mostly using Go language. It runs under privileged user as a system service/daemon.
Some of the features include: multiple protocols (OpenVPN, WireGuard), Kill-switch, Custom DNS and more.

This project is in use by **IVPN Client Desktop** project (*ivpn-desktop-ui*)

IVPN Client Desktop app is distributed on the official site [www.ivpn.net](www.ivpn.net).  

* [Installation](#installation)
* [Versioning](#versioning)
* [Contributing](#contributing)
* [Security Policy](#security)
* [License](#license)
* [Authors](#Authors)
* [Acknowledgements](#acknowledgements)

## Installation

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes.

### Requirements

#### Windows

  - Windows 10+
  - NSIS installer
  - Microsoft Visual Studio Community 2019 or Build Tools for Visual Studio 2019
    (ensure Windows SDK 10.0 is installed)
  - Go 1.13+ (downloads automatically by the build script)
  - WiX Toolset (downloads automatically by the build script)

#### macOS

  - macOS Mojave 10.14.6
  - Xcode Command Line Tools
  - Go 1.13+

### Compilation

#### Windows

To compile IVPN service binary run the batch file from the terminal. Use Developer Command Prompt for Visual Studio (required for building native sub-projects).
```
References/Windows/scripts/build-all.bat
```
The batch script will compile IVPN Service binary and all required dependencies.

#### macOS

To compile IVPN daemon binary run the batch file from the terminal.
```
References/macOS/scripts/build-all.sh
```
The batch script will compile IVPN Service binary and all required dependencies (OpenVPN, WireGuard).
Compiled binaries location:

  - WireGuard:  `References/macOS/_deps/wg_inst`
  - OpenVPN:  `References/macOS/_deps/openvpn_inst/bin`
  - IVPN Service: `IVPN Agent`

**Note!** In order to run application as macOS daemon, the binary must be signed by Apple Developer ID.

## Versioning

Project is using [Semantic Versioning (SemVer)](https://semver.org) for creating release versions.

SemVer is a 3-component system in the format of `x.y.z` where:

`x` stands for a **major** version  
`y` stands for a **minor** version  
`z` stands for a **patch**

So we have: `Major.Minor.Patch`

## Contributing

If you are interested in contributing to IVPN Daemon for IVPN Client Desktop project, please read our [Contributing Guidelines](/.github/CONTRIBUTING.md).

## Security Policy

If you want to report a security problem, please read our [Security Policy](/.github/SECURITY.md).

## License

This project is licensed under the GPLv3 - see the [License](/LICENSE.md) file for details.

## Authors

See the [Authors](/AUTHORS) file for the list of contributors who participated in this project.

## Acknowledgements

See the [Acknowledgements](/ACKNOWLEDGEMENTS.md) file for the list of third party libraries used in this project.
