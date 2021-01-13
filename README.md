# IVPN command line interface (CLI)

**IVPN command line interface** is an official IVPN command line client.  
It is a client for  IVPN daemon ([ivpn-desktop-daemon](https://github.com/ivpn/desktop-app-daemon))   
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

This is the official Git repo of the [IVPN command line interface](https://github.com/ivpn/desktop-app-cli).

<a name="installation"></a>
## Installation

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes.

### Requirements

#### Linux

  - [Go 1.13+](https://golang.org/)
  - [FPM](https://fpm.readthedocs.io/en/latest/installing.html)
  - `rpm` package to be able to compile RPM packages (use command `sudo apt-get install rpm` to install on Ubuntu)
  - Git
  - `curl` package (use command `sudo apt install curl` to install on Ubuntu)

#### macOS
  - [Go 1.13+](https://golang.org/)
  - Git

#### Windows
  - [Go 1.13+](https://golang.org/)
  - Git

### Compilation


#### macOS
  ```
  git clone https://github.com/ivpn/desktop-app-cli.git
  cd desktop-app-cli/
  ./References/macOS/build.sh
  ```
  Compiled binary can be found here:
  ```
  References/macOS/_out_bin/
  ```

#### Windows
  Ensure that GO111MODULE is enabled (`set GO111MODULE=on`)
  ```
  git clone https://github.com/ivpn/desktop-app-cli.git
  cd desktop-app-cli
  References\Windows\build.bat
  ```
  Compiled binaries can be found here:
  ```
  bin\x86\cli
  bin\x86_64\cli
  ```  

#### Linux
  The CLI binary compiles together with [desktop-app-daemon](https://github.com/ivpn/desktop-app-daemon) project. It required to be able to compile into redistributable packages (ready for installation DEB\RPM).
  Therefore, a few manipulations with GOPATH and folder structure necessary to perform.

  * Set `$GOPATH` variable to your projects folder  
      -  example: `export GOPATH=$HOME/Projects`  

  * Create folder structure in a projects folder `$GOPATH/src/github.com/ivpn`  
      -  `cd $GOPATH`  
      -  `mkdir -p src/github.com/ivpn`  

  * Clone CLI project repository  
      -  `cd $GOPATH/src/github.com/ivpn`  
      -  `git clone https://github.com/ivpn/desktop-app-cli.git`  

  *  Clone [desktop-app-daemon](https://github.com/ivpn/desktop-app-daemon) project
      -  `cd $GOPATH/src/github.com/ivpn`  
      -  `git clone https://github.com/ivpn/desktop-app-daemon.git`  
      
  *  Some packages required to be installed to be able to compile daemon sources.  
     Example of installing required packages for Ubuntu:
      ``` 
      #install 'libiw-dev' package
      sudo apt-get install libiw-dev

      #install 'rpm' package
      sudo apt install rpm
      ```

  * To compile projects and to build `.DEB` and `.RPM` packages just run `build-packages.sh` script:  
      -  `cd $GOPATH/src/github.com/ivpn/desktop-app-cli`  
      -  `References/Linux/build-packages.sh -v 2.12.8`  

Compiled packages can be found at `$GOPATH/src/github.com/ivpn/desktop-app-cli/References/Linux/_out_bin`  

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
