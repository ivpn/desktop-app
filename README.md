# IVPN command line interface (CLI)

**IVPN command line interface** is an official IVPN command line client.
It divided on two parts:
  - IVPN CLI (this repository)
  - IVPN service/daemon (repository [ivpn-desktop-daemon](https://github.com/ivpn/desktop-app-daemon))
Can be compiled for different platforms: Windows, macOS, Linux

IVPN CLI is distributed on the official site [www.ivpn.net](www.ivpn.net).  

* [Installation](#installation)
* [Versioning](#versioning)
* [Contributing](#contributing)
* [Security Policy](#security)
* [License](#license)
* [Authors](#Authors)
* [Acknowledgements](#acknowledgements)

<a name="installation"></a>
## Installation

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes.

### Requirements

#### Linux

  - [Go 1.13+](https://golang.org/)
  - [FPM](https://fpm.readthedocs.io/en/latest/installing.html)
  - rpm package to be able to compile RPM packages (use command `sudo apt-get install rpm` to install on Ubuntu)
  - Git
  - curl package (use command `sudo apt install curl` to install on Ubuntu)

#### macOS

#### Windows

### Compilation

#### Linux

  * set `$GOPATH` variable to your projects folder
    example: `export GOPATH=$HOME/Projects`
  * create folder structure in a projects folder `$GOPATH/src/github.com/ivpn`
    ```
    cd $GOPATH
    mkdir -p src/github.com/ivpn
    ```
  * clone CLI project repository
    ```
    cd $GOPATH/src/github.com/ivpn
    git clone https://github.com/ivpn/desktop-app-cli.git
    ```
  *  There is a dependency from a [Daemon for IVPN Client Desktop](https://github.com/ivpn/desktop-app-daemon) project. So, it is necessary to clone it's sources:
    ```
    cd $GOPATH/src/github.com/ivpn
    git clone https://github.com/ivpn/desktop-app-daemon.git
    ```
    An additional step required while the CLI project has beta status - switch to `feature/WC-903-Console-client` branch of daemon project:
    ```
    cd desktop-app-daemon
    git checkout feature/WC-903-Console-client
    ```

  * To compile projects and to build `.DEB` and `.RPM` packages just run `build-packages.sh` script:
    ```
    cd $GOPATH/src/github.com/ivpn/desktop-app-cli
    References/Linux/build-packages.sh -v 0.0.1
    ```
  Compiled packages can be found at `$GOPATH/src/github.com/ivpn/desktop-app-cli/References/Linux/_out_bin`

#### macOS

#### Windows

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
