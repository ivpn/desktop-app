# IVPN for Desktop (Windows/macOS/Linux)

[![CodeQL](https://github.com/ivpn/desktop-app/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/ivpn/desktop-app/actions/workflows/codeql-analysis.yml)
[![Security Scan (gosec)](https://github.com/ivpn/desktop-app/actions/workflows/gosec.yml/badge.svg)](https://github.com/ivpn/desktop-app/actions/workflows/gosec.yml)
[![CI](https://github.com/ivpn/desktop-app/actions/workflows/ci.yml/badge.svg)](https://github.com/ivpn/desktop-app/actions/workflows/ci.yml)
[![ivpn](https://snapcraft.io/ivpn/badge.svg)](https://snapcraft.io/ivpn)

**IVPN for Desktop** is the official IVPN app for desktop platforms. Some of the features include: multiple protocols (OpenVPN, WireGuard), Kill-switch, Multi-Hop, Trusted Networks, AntiTracker, Custom DNS, Dark mode, and more.  
IVPN Client app is distributed on the official site [www.ivpn.net](https://www.ivpn.net).

![IVPN application image](/.github/readme_images/ivpn_app.png#gh-light-mode-only)
![IVPN application image](/.github/readme_images/ivpn_app_dark.png#gh-dark-mode-only)

* [About this Repo](#about-repo)
* [Installation](#installation)
  * [Requirements](#requirements)
    * [Windows](#requirements_windows)
    * [macOS](#requirements_macos)
    * [Linux](#requirements_linux)
  * [Compilation](#compilation)
    * [Windows](#compilation_windows)
    * [macOS](#compilation_macos)
    * [Linux](#compilation_linux)
* [Versioning](#versioning)
* [Contributing](#contributing)
* [Security Policy](#security)
* [License](#license)
* [Authors](#Authors)
* [Acknowledgements](#acknowledgements)

<a name="about-repo"></a>

## About this Repo

This is the official Git repo of the [IVPN for Desktop](https://github.com/ivpn/desktop-app) app.

The project is divided into three parts:  

* **daemon**: Core module of the IVPN software built mostly using the Go language. It runs with privileged rights as a system service/daemon.  
* **UI**: Graphical User Interface built using Electron.  
* **CLI**: Command Line Interface.  

<a name="installation"></a>

## Installation

These instructions enable you to get the project up and running on your local machine for development and testing purposes.

<a name="requirements"></a>

### Requirements

<a name="requirements_windows"></a>

#### Windows

[Go 1.21+](https://golang.org/); Git; [npm](https://www.npmjs.com/get-npm); [Node.js (18)](https://nodejs.org/); [nsis3](https://nsis.sourceforge.io/Download); Build Tools for Visual Studio 2019 ('Windows 10 SDK 10.0.19041.0', 'Windows 11 SDK 10.0.22000.0', 'MSVC v142 C++ x64 build tools', 'C++ ATL for latest v142 build tools'); gcc compiler (e.g. [TDM GCC](https://jmeubank.github.io/tdm-gcc/download/)).  

<a name="requirements_macos"></a>

#### macOS

[Go 1.21+](https://golang.org/); Git; [npm](https://www.npmjs.com/get-npm); [Node.js (18)](https://nodejs.org/); Xcode Command Line Tools.  
To compile the OpenVPN/OpenSSL binaries locally, additional packages are required:  
```bash
brew install autoconf automake libtool
```
To compile  [liboqs](https://github.com/open-quantum-safe/liboqs), additional packages are required:  

```bash
brew install cmake ninja openssl@1.1 wget doxygen graphviz astyle valgrind
pip3 install pytest pytest-xdist pyyaml
```

<a name="requirements_linux"></a>

#### Linux

[Go 1.21+](https://golang.org/); Git; [npm](https://www.npmjs.com/get-npm); [Node.js (18)](https://nodejs.org/); gcc; make; [FPM](https://fpm.readthedocs.io/en/latest/installation.html); curl; rpm; libiw-dev.  

To compile  [liboqs](https://github.com/open-quantum-safe/liboqs), additional packages are required:  
`sudo apt install astyle cmake gcc ninja-build libssl-dev python3-pytest python3-pytest-xdist unzip xsltproc doxygen graphviz python3-yaml valgrind`

<a name="compilation"></a>

### Compilation

<a name="compilation_windows"></a>

#### Windows

Instructions to build installer of IVPN Client *(daemon + CLI + UI)*:  
Use Developer Command Prompt for Visual Studio (required for building native sub-projects).  

```bash
git clone https://github.com/ivpn/desktop-app.git
cd desktop-app/ui/References/Windows
build.bat
```

  Compiled binaries can be found at: `ui/References/Windows/bin`  

<a name="compilation_macos"></a>

#### macOS

Instructions to build DMG package of IVPN Client *(daemon + CLI + UI)*:  

```bash
git clone https://github.com/ivpn/desktop-app.git
cd desktop-app/ui/References/macOS
./build.sh -v <VERSION_X.X.X> -c <APPLE_DevID_CERTIFICATE>
```

Compiled binary can be found at: `ui/References/macOS/_compiled`  
*([some info](https://github.com/ivpn/desktop-app/issues/161) about Apple Developer ID)*  

<a name="compilation_linux"></a>

#### Linux

```bash
# get sources
git clone https://github.com/ivpn/desktop-app.git
cd desktop-app
```

Base package *(daemon + CLI)*:

```bash
./cli/References/Linux/build.sh
```

Compiled DEB/RPM packages can be found at `cli/References/Linux/_out_bin`  
*Note: You can refer to [manual installation guide for Linux](docs/readme-build-manual.md).*

Graphical User Interface *(UI)*:

```bash
./ui/References/Linux/build.sh
```

Compiled DEB/RPM packages can be found at `ui/References/Linux/_out_bin`  
*Note: It is required to have installed IVPN Daemon before running IVPN UI.*  

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

If you are interested in contributing to IVPN for Desktop project, please read our [Contributing Guidelines](/.github/CONTRIBUTING.md).

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
