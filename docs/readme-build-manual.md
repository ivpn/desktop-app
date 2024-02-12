# Manual build and installation

This document describes how to manually build and install IVPN binaries on Linux.

## Requirements

Please, refer to development dependencies described in the [main readme](../readme.md#requirements_linux).

## Compilation

Clone the sources and build everything using the script `cli/References/Linux/build.sh`:

```bash
git clone https://github.com/ivpn/desktop-app.git
cd desktop-app/cli/References/Linux/
./build.sh
```

As a result, you will have compiled all the required binaries:  

- service binary  (`daemon/References/Linux/scripts/_out_bin/ivpn-service`)  
- CLI binary (`cli/References/Linux/_out_bin/ivpn`)
- all third-party binaries (located under folders `daemon\References\Linux\_deps\*_inst`)
- ready-to-use DEB and RPM packages (located at `cli/References/Linux/_out_bin`)

## Manual installation

Manual installation involves placing compiled binaries into specified locations on the target system. The IVPN service checks for the existence of required files, their access rights, and owner. All the necessary files should be located under `/opt/ivpn`.

Below is an example of a correct installation.  
The file source locations are indicated as comments (e.g. `# path-relative-to-repository-root`)  
***Note: Files' owner and access rights are important!***  

```bash
/opt/ivpn/etc:
-r-------- 1 root root  2358 Feb  8 16:10 ca.crt            # daemon\References\common\etc
-rwx------ 1 root root   268 Feb  8 16:10 client.down       # daemon\References\Linux\etc
-rwx------ 1 root root  2664 Feb  8 16:10 client.up         # daemon\References\Linux\etc
-r-------- 1 root root 28111 Feb  8 16:10 dnscrypt-proxy-template.toml # daemon\References\common\etc
-rwx------ 1 root root 27168 Feb  8 16:10 firewall.sh       # daemon\References\Linux\etc
-rw------- 1 root root 68694 Feb  8 16:10 servers.json      # daemon\References\common\etc
-rwx------ 1 root root 33173 Feb  8 16:10 splittun.sh       # daemon\References\Linux\etc
-r-------- 1 root root   636 Feb  8 16:10 ta.key            # daemon\References\common\etc

/opt/ivpn/dnscrypt-proxy:
-rwxr-xr-x 1 root root 10828056 Feb  8 16:10 dnscrypt-proxy # daemon\References\Linux\_deps\dnscryptproxy_inst

/opt/ivpn/kem:
-rwxr-xr-x 1 root root 313568 Feb  8 16:10 kem-helper   # daemon\References\Linux\_deps\kem-helper\kem-helper-bin

/opt/ivpn/obfsproxy:
-rwxr-xr-x 1 root root 5460792 Feb  8 16:10 obfs4proxy  # daemon\References\Linux\_deps\obfs4proxy_inst

/opt/ivpn/v2ray:
-rwxr-xr-x 1 root root 31774552 Feb  8 16:10 v2ray      # daemon\References\Linux\_deps\v2ray_inst

/opt/ivpn/wireguard-tools:
-rwxr-xr-x 1 root root 101312 Feb  8 16:10 wg           # daemon\References\Linux\_deps\wireguard-tools_inst
-rwxr-xr-x 1 root root  13460 Feb  8 16:10 wg-quick     # daemon\References\Linux\_deps\wireguard-tools_inst
```

Place the IVPN binaries in the system's binary folder to access them from the terminal:

 ```bash
 /usr/bin/ivpn-service  # service: `daemon/References/Linux/scripts/_out_bin/ivpn-service`
 /usr/bin/ivpn          # CLI binary: `cli/References/Linux/_out_bin/ivpn`
 ```

The IVPN service must be started under a privileged user.  
You can use the command-line parameter `--logging` to force-enable logging for the service.

*Example:*  

```bash
# Run IVPN service
/usr/bin/ivpn-service --logging
# Run the IVPN CLI (from a separate terminal!)
# The CLI requires the IVPN service to be already started
/usr/bin/ivpn
```

You can configure your environment to start the ivpn-service as a daemon.  

#### Example: Register and start as systemd daemon

Example of a `systemd` configuration file (`/etc/systemd/system/ivpn-service.service`):  

```bash
[Unit]
Description=ivpn-service

[Service]
Type=simple
User=root
Group=root
ExecStart=/usr/bin/ivpn-service 
Restart=always
WorkingDirectory=/
TimeoutStopSec=infinity

[Install]
WantedBy=multi-user.target
```

Start `systemd` service example:

```bash
# reload the systemd daemon to recognize your new service (`/etc/systemd/system/ivpn-service.service`)
sudo systemctl daemon-reload 
# start service
sudo systemctl start ivpn-service
# enable service to start automatically at boot
sudo systemctl enable ivpn-service
```


## Graphical User Interface 

Compilation:

```bash
cd desktop-app/ui/References/Linux
./build.sh
```

Compiled files location:

- compiled binaries: `ui/dist/bin`  
- ready-to-use DEB/RPM packages: `ui/References/Linux/_out_bin`  

## Useful links  

- [ArchLinux User Repository installation files (base package)](https://aur.archlinux.org/cgit/aur.git/tree/?h=ivpn)
- [ArchLinux User Repository installation files (UI package)](https://aur.archlinux.org/cgit/aur.git/tree/?h=ivpn-ui)
- [Configuring IVPN daemon on OpenRC init systems](https://github.com/ivpn/desktop-app/issues/1#issuecomment-822919358)  
