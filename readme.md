# Daemon for IVPN Client Desktop (Windows/macOS/Linux)

**Daemon for IVPN Client Desktop** is a core module of IVPN Client for Windows, macOS and Linux built mostly using Go language. It runs under privileged user as a system service/daemon.
Some of the features include: multiple protocols (OpenVPN, WireGuard), Kill-switch, Custom DNS and more.

This project is in use by [IVPN Client Desktop](https://github.com/ivpn/desktop-app-ui) and [IVPN command line interface](https://github.com/ivpn/desktop-app-cli) projects.

IVPN Client Desktop app is distributed on the official site [www.ivpn.net](https://www.ivpn.net).  

* [About this Repo](#about-repo)
* [Installation](#installation)
* [Versioning](#versioning)
* [Contributing](#contributing)
* [Security Policy](#security)
* [License](#license)
* [Authors](#Authors)
* [Acknowledgements](#acknowledgements)

<a name="about-repo"></a>
## About this Repo

This is the official Git repo of the [Daemon for IVPN Client Desktop](https://github.com/ivpn/desktop-app-daemon).


<a name="installation"></a>
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

<a name="versioning"></a>
## Versioning

Project is using [Semantic Versioning (SemVer)](https://semver.org) for creating release versions.

SemVer is a 3-component system in the format of `x.y.z` where:

`x` stands for a **major** version  
`y` stands for a **minor** version  
`z` stands for a **patch**

So we have: `Major.Minor.Patch`

<a name="contributing"></a>
## Contributing

If you are interested in contributing to IVPN Daemon for IVPN Client Desktop project, please read our [Contributing Guidelines](/.github/CONTRIBUTING.md).

<a name="security"></a>
## Security Policy

If you want to report a security problem, please read our [Security Policy](/.github/SECURITY.md).

<a name="license"></a>
## License

This project is licensed under the GPLv3 - see the [License](/LICENSE.md) file for details.

<a name="Authors"></a>
## Authors

See the [Authors](/AUTHORS) file for the list of contributors who participated in this project.

<a name="acknowledgements"></a>
## Acknowledgements

See the [Acknowledgements](/ACKNOWLEDGEMENTS.md) file for the list of third party libraries used in this project.
