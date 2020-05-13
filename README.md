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
 
  * To compile projects and to build `.DEB` and `.RPM` packages just run `build-packages.sh` script:  
      -  `cd $GOPATH/src/github.com/ivpn/desktop-app-cli`  
      -  `References/Linux/build-packages.sh -v 0.0.1`  

Compiled packages can be found at `$GOPATH/src/github.com/ivpn/desktop-app-cli/References/Linux/_out_bin`  

#### macOS
  ```
  git clone https://github.com/ivpn/desktop-app-cli.git
  cd desktop-app-cli/
  ./References/macOS/build.sh 
  ``` 
  Comliled binariy can be found here:
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
  Comliled binaries can be found here:
  ```
  bin\x86\cli
  bin\x86_64\cli
  ```

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
