# IVPN Daemon (Windows/macOS/Linux)

**IVPN Daemon** is a core module of IVPN Client for desktop platforms (Windows/macOS/Linux) built mostly using Go language. It runs under privileged user as a system service/daemon.  
Some of the features include:
  - multiple protocols (OpenVPN, WireGuard)
  - Kill-switch
  - custom DNS
  - Multi-Hop
  - AntiTracker

This project is in use by [IVPN Client UI](https://github.com/ivpn/desktop-app-ui) and [IVPN CLI](https://github.com/ivpn/desktop-app-cli) projects.

IVPN Client app is distributed on the official site [www.ivpn.net](https://www.ivpn.net).  

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

This is the official Git repo of the [IVPN Daemon](https://github.com/ivpn/desktop-app-daemon).


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
  - Git

#### macOS

  - macOS Mojave 10.14.6
  - Xcode Command Line Tools
  - Go 1.13+
  - Git

#### Linux
  - Go 1.13+
  - Git

### Compilation

#### Windows

To compile IVPN service binary run the batch file from the terminal. Use Developer Command Prompt for Visual Studio (required for building native sub-projects).
```
git clone https://github.com/ivpn/desktop-app-daemon.git
cd desktop-app-daemon
References/Windows/scripts/build-all.bat
```
The batch script will compile IVPN Service binary and all required dependencies.

#### macOS

To compile IVPN daemon binary run the batch file from the terminal.
```
git clone https://github.com/ivpn/desktop-app-daemon.git
cd desktop-app-daemon
References/macOS/scripts/build-all.sh
```
The batch script will compile IVPN Service binary and all required dependencies (OpenVPN, WireGuard).
Compiled binaries location:

  - WireGuard:  `References/macOS/_deps/wg_inst`
  - OpenVPN:  `References/macOS/_deps/openvpn_inst/bin`
  - IVPN Service: `IVPN Agent`

**Note!** In order to run application as macOS daemon, the binary must be signed by Apple Developer ID.

#### Linux
Some packages required to be installed to be able to compile daemon sources.
Example of installing required packages for Ubuntu:
``` 
#install 'libiw-dev' package
sudo apt-get install libiw-dev

#install 'rpm' package
sudo apt install rpm
```

Run build script:
```
git clone https://github.com/ivpn/desktop-app-daemon.git
cd desktop-app-daemon
./References/Linux/scripts/build-all.sh  
```
The compiled binary can be found at `References/Linux/scripts/_out_bin`

**Note!**
IVPN Daemon must be installed appropriately on a target system. We recommend referring the [IVPN CLI](https://github.com/ivpn/desktop-app-cli) project for the build instructions to compile IVPN redistributable packages for Linux.

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
