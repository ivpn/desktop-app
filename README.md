# IVPN Command Line Interface (CLI)

**IVPN Command Line Interface** is an official CLI for IVPN software.  
It is a client for IVPN daemon ([ivpn-desktop-daemon](https://github.com/ivpn/desktop-app-daemon))   
Can be compiled for different platforms: Windows, macOS, Linux

IVPN CLI is distributed on the official site [https://www.ivpn.net](https://www.ivpn.net).  

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

This is the official Git repo of the [IVPN Command Line Interface](https://github.com/ivpn/desktop-app-cli).

<a name="installation"></a>
## Installation

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes.

### Requirements


#### Windows
  - [Go 1.13+](https://golang.org/)
  - Git
  
#### macOS
  - [Go 1.13+](https://golang.org/)
  - Git
  
#### Linux
  - [Go 1.13+](https://golang.org/)
  - packages: [FPM](https://fpm.readthedocs.io/en/latest/installing.html), curl, rpm, libiw-dev
  - Git

### Compilation

#### Windows

  **Note!**
  IVPN Daemon must be installed appropriately on a target system.  
  We recommend using [IVPN Client UI](https://github.com/ivpn/desktop-app-ui2) project to build a Windows installer for IVPN software.  
  
```
git clone https://github.com/ivpn/desktop-app-cli.git
cd desktop-app-cli
References\Windows\build.bat <VERSION_X.X.X>
```

  Compiled binaries can be found at: `bin\x86_64\cli`  
  
#### macOS
  
  **Note!**
  IVPN Daemon must be installed appropriately on a target system.  
  We recommend using [IVPN Client UI](https://github.com/ivpn/desktop-app-ui2) project to build a macOS DMG package for IVPN software.  
  
```
git clone https://github.com/ivpn/desktop-app-cli.git
cd desktop-app-cli/
./References/macOS/build.sh -v <VERSION_X.X.X>
```

  Compiled binary can be found at: `References/macOS/_out_bin/`

#### Linux
    
```
git clone https://github.com/ivpn/desktop-app-daemon.git
git clone https://github.com/ivpn/desktop-app-cli.git
cd desktop-app-cli/References/Linux/
./build.sh -v <VERSION_X.X.X>
```
  
  Compiled packages can be found at `desktop-app-cli/References/Linux/_out_bin`  
  
  **Info**
  You may be interested also in [IVPN Client UI](https://github.com/ivpn/desktop-app-ui2) project to build a 'UI' Linux redistributable packages of IVPN software.
  
  
##### Manual installation on Linux
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
-r-------- 1 root root   451 May 25 16:50 signing.pub
-r-------- 1 root root   636 May 25 16:50 ta.key
```
1. Build the current project to get 'ivpn service' and 'ivpn cli' binaries.
2. Create folder `/opt/ivpn/etc`
3. Copy all required files (see above).  
    **Note!** Files owner and access rights are important!
4. Now you can start compiled service binary from the command line (just to check if it works).  
    **Note!** The service must be started under a privileged user!  
    **Info!** You can use the command line parameter `--logging` to enable logging for service.  
    4.1. Simply run compiled ivpn-cli binary to check if it successfully connects to the service (use separate terminal).
5. If everything works - you can configure your environment to start ivpn-service automatically with the system boot (we are using systemd for such purposes)

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

If you are interested in contributing to IVPN CLI project, please read our [Contributing Guidelines](/.github/CONTRIBUTING.md).

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
