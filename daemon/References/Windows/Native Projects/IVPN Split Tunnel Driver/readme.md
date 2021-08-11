# IVPN Split-Tunnel Driver for Windows

<a name="about"></a>
## About
Provides functionality to split traffic for specific applications (binaries).
All the traffic for configured binaries will be excluded from the VPN tunnel. All traffic of child processes started by configured binaries will be excluded also.

<a name="installation"></a>
## Installation
These instructions enable you to get the project up and running on your local machine for development and testing purposes.

<a name="requirements"></a>
### Requirements
[Visual Studio 2019 and Windows Driver Development Kit](https://docs.microsoft.com/en-us/windows-hardware/drivers/download-the-wdk)

<a name="preparations"></a>
### Preparations

1) [Install VS + DDK](https://docs.microsoft.com/en-us/windows-hardware/drivers/download-the-wdk)
2) (optional; needed for possibility to debug driver) [Provision a computer for driver deployment and testing](https://docs.microsoft.com/en-us/windows-hardware/drivers/gettingstarted/provision-a-target-computer-wdk-8-1)

<a name="compilation"></a>
### Compilation
`build.bat <EV_Certificate_SHA1_hash>`

<a name="testing"></a>
### Testing
The console application project to play with the driver:
`others\SplitTunTestConsole\SplitTunTestConsole.vcxproj`
It can control all aspects of driver functionality.

<a name="useful_links"></a>
## Useful links
https://docs.microsoft.com/en-us/windows-hardware/drivers/wdf/
https://docs.microsoft.com/en-us/windows-hardware/drivers/wdf/using-kernel-mode-driver-framework-with-non-pnp-drivers
https://docs.microsoft.com/en-us/windows-hardware/drivers/network/inf-files-for-callout-drivers
https://docs.microsoft.com/en-us/windows-hardware/drivers/debugger/debug-universal-drivers---step-by-step-lab--echo-kernel-mode-
https://docs.microsoft.com/en-us/windows-hardware/drivers/gettingstarted/writing-a-very-small-kmdf--driver
https://docs.microsoft.com/en-us/windows-hardware/drivers/devtest/adding-wpp-software-tracing-to-a-windows-driver
https://docs.microsoft.com/en-us/windows-hardware/drivers/network/using-bind-or-connect-redirection
https://docs.microsoft.com/en-us/windows-hardware/drivers/network/data-field-identifiers
https://docs.microsoft.com/en-us/windows-hardware/drivers/ddi/fwpsk/ne-fwpsk-fwps_fields_ale_connect_redirect_v4_
https://docs.microsoft.com/en-us/windows-hardware/drivers/network/processing-classify-callouts
https://docs.microsoft.com/en-us/windows-hardware/drivers/network/filtering-layer
https://docs.microsoft.com/en-us/windows-hardware/drivers/network/filtering-condition-flags
