# IVPN command line interface (CLI)

**IVPN command line interface** is an official IVPN command line client.
It divided on two parts:
  - IVPN CLI (this repository)
  - IVPN service/daemon (repository *ivpn-desktop-daemon*)
Can be compiled for different platforms: Windows, macOS, Linux

IVPN CLI is distributed on the official site [www.ivpn.net](www.ivpn.net).  

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
  - Go 1.13+

### Compilation

There is a dependency from **IVPN Daemon for IVPN Client Desktop** sources. So, it is necessary to have it's sources:

  - please, checkout *ivpn-desktop-daemon* repository.
  - update path to *ivpn-desktop-daemon* location in file `References/config/daemon_repo_local_path.txt` (if necessary)

#### Linux

  To build ".deb" and ".rpm" packages just run script:
  ```
  References/Linux/build-packages.sh -v 0.0.1
  ```
  Compiled packages can be found at `References/Linux/_out_bin`

#### macOS

#### Windows

## Versioning

Project is using [Semantic Versioning (SemVer)](https://semver.org) for creating release versions.

SemVer is a 3-component system in the format of `x.y.z` where:

`x` stands for a **major** version  
`y` stands for a **minor** version  
`z` stands for a **patch**

So we have: `Major.Minor.Patch`

## Contributing

If you are interested in contributing to IVPN CLI project, please read our [Contributing Guidelines](/.github/CONTRIBUTING.md).

## Security Policy

If you want to report a security problem, please read our [Security Policy](/.github/SECURITY.md).

## License

This project is licensed under the GPLv3 - see the [License](/LICENSE.md) file for details.

## Authors

See the [Authors](/AUTHORS) file for the list of contributors who participated in this project.

## Acknowledgements

See the [Acknowledgements](/ACKNOWLEDGEMENTS.md) file for the list of third party libraries used in this project.
