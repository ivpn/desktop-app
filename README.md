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
  [npm](https://www.npmjs.com/get-npm)


### Compilation

Update all project  dependencies:
```
npm install
```

Compile binary for current platform:
```
npm run electron:build
```

The compiled binary will be available in the folder `dist_electron`.

**Important:** To be able to run the compiled UI app, the latest IVPN Client version must be installed.
[IVPN official apps](https://www.ivpn.net/apps-overview).
[A daemon sources](https://github.com/ivpn/desktop-app-daemon).
[An IVPN Client sources](https://github.com/ivpn/desktop-app-ui).

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
