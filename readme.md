[![CodeQL](https://github.com/ivpn/desktop-app/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/ivpn/desktop-app/actions/workflows/codeql-analysis.yml)
[![Security Scan (gosec)](https://github.com/ivpn/desktop-app/actions/workflows/gosec.yml/badge.svg)](https://github.com/ivpn/desktop-app/actions/workflows/gosec.yml)
[![CI](https://github.com/ivpn/desktop-app/actions/workflows/ci.yml/badge.svg)](https://github.com/ivpn/desktop-app/actions/workflows/ci.yml)
[![ivpn](https://snapcraft.io/ivpn/badge.svg)](https://snapcraft.io/ivpn)
# IVPN for Desktop (Windows/macOS/Linux)

**IVPN for Desktop** is the official IVPN app for desktop platforms. Some of the features include: multiple protocols (OpenVPN, WireGuard), Kill-switch, Multi-Hop, Trusted Networks, AntiTracker, Custom DNS, Dark mode, and more.  
IVPN Client app is distributed on the official site [www.ivpn.net](https://www.ivpn.net).  
![IVPN application image](/.github/readme_images/ivpn_app.png)
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
      * [Linux Daemon](#compilation_linux_daemon)
        * [Manual installation IVPN daemon on Linux](#compilation_linux_daemon_manual_install)
      * [Linux UI](#compilation_linux_ui)
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
- **Daemon** is a core module of IVPN software built mostly using the Go language. It runs with privileged rights as a system service/daemon.  
- **UI** is a Graphical User Interface built using Electron.  
- **CLI** is a Command Line Interface.  

<a name="installation"></a>
## Installation

These instructions enable you to get the project up and running on your local machine for development and testing purposes.

<a name="requirements"></a>
### Requirements

<a name="requirements_windows"></a>
#### Windows

[npm](https://www.npmjs.com/get-npm); [Node.js (LTS version)](https://nodejs.org/); [nsis2](https://nsis.sourceforge.io/Download); Build Tools for Visual Studio 2019 ('Windows 10 SDK 10.0.19041.0', 'Windows 11 SDK 10.0.22000.0', 'MSVC v142 C++ x64 build tools', 'C++ ATL for latest v142 build tools'); gcc compiler e.g. [TDM GCC](https://jmeubank.github.io/tdm-gcc/download/); [Go 1.18+](https://golang.org/); Git

<a name="requirements_macos"></a>
#### macOS

[npm](https://www.npmjs.com/get-npm); [Node.js (LTS version)](https://nodejs.org/); Xcode Command Line Tools; [Go 1.18+](https://golang.org/); Git  
To compile the OpenVPN\OpenSSL binaries locally, additional packages are needed: `brew install autoconf automake libtool`

<a name="requirements_linux"></a>
#### Linux
[npm](https://www.npmjs.com/get-npm); [Node.js (LTS version)](https://nodejs.org/); packages: [FPM](https://fpm.readthedocs.io/en/latest/installation.html), curl, rpm, libiw-dev; [Go 1.18+](https://golang.org/); Git

<a name="compilation"></a>
### Compilation

<a name="compilation_windows"></a>
#### Windows
Instructions to build Windows installer of IVPN Client software (daemon+CLI+UI):  
Use Developer Command Prompt for Visual Studio (required for building native sub-projects).  

```
git clone https://github.com/ivpn/desktop-app.git
cd desktop-app/ui/References/Windows
build.bat
```

  Compiled binaries can be found at: `desktop-app/ui/References/Windows/bin`  

<a name="compilation_macos"></a>
#### macOS
Instructions to build macOS DMG package of IVPN Client software (daemon+CLI+UI):  

```
git clone https://github.com/ivpn/desktop-app.git
cd ivpn/desktop-app/ui/References/macOS
./build.sh -v <VERSION_X.X.X> -c <APPLE_DevID_CERTIFICATE>
```

  Compiled binary can be found at: `desktop-app/ui/References/macOS/_compiled`  
  *([some info](https://github.com/ivpn/desktop-app/issues/161) about Apple Developer ID)*  

<a name="compilation_linux"></a>
#### Linux

<a name="compilation_linux_daemon"></a>
##### Linux Daemon

Instructions to build Linux DEB and RPM packages of IVPN software ('base' package: daemon + CLI):  

```
git clone https://github.com/ivpn/desktop-app.git
cd desktop-app/cli/References/Linux/
./build.sh -v <VERSION_X.X.X>
```

  Compiled packages can be found at `desktop-app/cli/References/Linux/_out_bin`  

<a name="compilation_linux_daemon_manual_install"></a>
###### Manual installation IVPN daemon on Linux
Sometimes it is required to have the possibility to install IVPN binaries manually.  
It's easy to do it by following the rules described below.

The ivpn-service is checking the existing of some required files (all files can be found in the repository)
```
VirtualBox:/opt/ivpn/etc$ ls -l
total 52
-r-------- 1 root root  2358 May 25 16:50 ca.crt
-rwx------ 1 root root   113 May 25 16:50 client.down
-rwx------ 1 root root  1927 May 25 16:50 client.up
-rwx------ 1 root root  5224 May 25 16:50 firewall.sh
-rw------- 1 root root 21524 May 26 20:52 servers.json
-r-------- 1 root root   636 May 25 16:50 ta.key
```
1. Build the current project to get 'ivpn service' and 'ivpn cli' binaries.
2. Create folder `/opt/ivpn/etc`
3. Copy all required files (see above).  
    **Note!** Files owner and access rights are important.
4. Now you can start compiled service binary from the command line (just to check if it works).  
    **Note!** The service must be started under a privileged user.  
    **Info** You can use the command line parameter `--logging` to enable logging for service.  
    4.1. Simply run compiled ivpn-cli binary to check if it successfully connects to the service (use separate terminal).
5. If everything works - you can configure your environment to start ivpn-service automatically with the system boot (we are using systemd for such purposes)

<a name="compilation_linux_ui"></a>
##### Linux UI
Instructions to build Linux DEB and RPM packages of IVPN software 'UI' package:  

```
git clone https://github.com/ivpn/desktop-app.git
cd desktop-app/ui/References/Linux
./build.sh -v <VERSION_X.X.X>
```

  Compiled packages can be found at `desktop-app-ui2/References/Linux/_out_bin`  

  **Note!**
  It is required to have installed IVPN Daemon before running IVPN UI.  

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
