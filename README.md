# IVPN Client UI (beta)

**IVPN Client UI  (beta)** is a multi-platform UI for IVPN Client Desktop ([ivpn-desktop-daemon](https://github.com/ivpn/desktop-app-daemon)) built using [Electron](https://www.electronjs.org/) (supported platforms: macOS, Linux, Windows).

IVPN Client app is distributed on the official site [www.ivpn.net](https://www.ivpn.net).  

* [About this Repo](#about-repo)
* [Installation](#installation)
* [Versioning](#versioning)
* [Contributing](#contributing)
* [Security Policy](#security)
* [License](#license)
* [Authors](#Authors)

<a name="about-repo"></a>
## About this Repo

This is the official Git repo of the [IVPN Client UI (beta)](https://github.com/ivpn/desktop-app-ui-beta).

<a name="installation"></a>
## Installation

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes.

### Requirements
  
#### Windows

  - [npm](https://www.npmjs.com/get-npm)
  - Build Tools for Visual Studio 2019 ('Windows SDK 10.0', 'MSVC v142 C++ x64 build tools', 'C++ ATL for latest v142 build tools')
  - Go 1.13+ (downloads automatically by the build script)
  - Git

#### macOS

  - [npm](https://www.npmjs.com/get-npm)
  - Xcode Command Line Tools
  - Go 1.13+
  - Git

#### Linux
  - [npm](https://www.npmjs.com/get-npm)
  - packages: [FPM](https://fpm.readthedocs.io/en/latest/installing.html), curl, rpm, libiw-dev
  - Go 1.13+
  - Git


### Compilation

#### Windows
Instructions to build Windows installer of IVPN Client software (daemon+CLI+UI):  

```
git clone https://github.com/ivpn/desktop-app-daemon.git
git clone https://github.com/ivpn/desktop-app-cli.git
git clone https://github.com/ivpn/desktop-app-ui2.git
cd desktop-app-ui2/References/Windows
build.bat
```

  Compiled binaries can be found at: `desktop-app-ui2\References\Windows\bin`  
  
#### macOS
Instructions to build macOS DMG package of IVPN Client software (daemon+CLI+UI):  
  
```
git clone https://github.com/ivpn/desktop-app-daemon.git
git clone https://github.com/ivpn/desktop-app-cli.git
git clone https://github.com/ivpn/desktop-app-ui2.git
cd ivpn/desktop-app-ui2/References/macOS
./build.sh -v <VERSION_X.X.X> -c <APPLE_DevID_CERTIFICATE>
```

  Compiled binary can be found at: `desktop-app-ui2/References/macOS/_compiled`

#### Linux
Instructions to build Linux DEB and RPM packages of IVPN software 'UI' package:  
    
```
git clone https://github.com/ivpn/desktop-app-ui2.git
cd desktop-app-ui2/References/Linux
./build.sh -v <VERSION_X.X.X>
```
  
  Compiled packages can be found at `desktop-app-ui2/References/Linux/_out_bin`  
  
  **Note!**
  It is required to have installed IVPN Daemon before running IVPN UI.  
  **Info:**
  You may be interested also in [IVPN Client CLI](https://github.com/ivpn/desktop-app-cli) project to build a 'base' Linux redistributable package (daemon + CLI) of IVPN software.

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

If you are interested in contributing to IVPN Client project, please read our [Contributing Guidelines](/.github/CONTRIBUTING.md).

<a name="security"></a>
## Security Policy

If you want to report a security problem, please read our [Security Policy](/.github/SECURITY.md).

<a name="license"></a>
## License

This project is licensed under the GPLv3 - see the [License](/LICENSE.md) file for details.

<a name="Authors"></a>
## Authors

See the [Authors](/AUTHORS) file for the list of contributors who participated in this project.
