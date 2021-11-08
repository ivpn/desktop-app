# Changelog

All notable changes to this project will be documented in this file.  


## Version 3.4.0 - 2021-11-08

[NEW] Multi-Hop for WireGuard protocol  
[NEW] Option to reset app settings on logout  
[NEW] Option to keep Firewall state on logout  
[NEW] CLI option to show all servers and to connect to specific server  
[NEW] (Linux) Obfsproxy now works on Linux  
[IMPROVED] Speed up the response timeout to API server  
[IMPROVED] Force automatic WireGuard key regeneration if the rotation interval has passed  
[IMPROVED] (Windows) Updated WireGuard: v0.4.9  
[IMPROVED] (Windows) Updated: OpenVPN: v2.5.3; OpenSSL: 1.1.1k  
[IMPROVED] (macOS) Updated: OpenVPN: v2.5.3; OpenSSL: 1.1.1k  
[IMPROVED] (macOS) Updated WireGuard binaries: wireguard-go: v0.0.20210424; wireguard-tools v1.0.20210914  
[IMPROVED] (Linux) WireGuard-tools integrated into a package (for a kernel since 5.6, no dependencies required to use WireGuard)  
[FIX] Fastest server settings were ignored in some cases  
[FIX] Option to run multiple UI instances in some cases  
[FIX] Server selection issues  
[FIX] Other minor issues and improvements  
[FIX] (Windows) Compatibility with Windows Server  
[FIX] (Windows) IVPN Firewall rules overlap blocking rules from Windows Firewall  
[FIX] (Windows) Icons created in %temp% each time app is launched  
[FIX] (macOS) Unable to start WireGuard connection if more than 10 utunX devices configured  
[FIX] (Linux) "Allow LAN traffic" does not persist after a restart  
[FIX] (Linux) UI crash after some Linux distribution updates  

[Download IVPN Client for Windows](https://repo.ivpn.net/windows/bin/IVPN-Client-v3.4.0.exe)  
SHA256: 01d876ad506ccf9def6c8ded2c104b740bb3093d728ad52168aecf597113f7d4   

[Download IVPN Client for macOS (Intel)](https://repo.ivpn.net/macos/bin/IVPN-3.4.0.dmg)  
SHA256: ca9d45f7df2eb95fa5f57ada9012d6add95113635b74f21df36c40725687b3f2  
[Download IVPN Client for macOS (M1)](https://repo.ivpn.net/macos/bin/IVPN-3.4.0-arm64.dmg)  
SHA256: 8a1f4bb2c01f289b2ca241b86c0b5eec4b1225de06777d076d2ef534e20e7481  

[Download IVPN Client for Linux (DEB)](https://repo.ivpn.net/stable/pool/ivpn_3.4.0_amd64.deb)  
SHA256: fad328c95679c983d162d117e909c4c0b5eacd7b5dd54b8de7e1a1c4dbeca64c   
[Download IVPN Client for Linux (RPM)](https://repo.ivpn.net/stable/pool/ivpn-3.4.0-1.x86_64.rpm)  
SHA256: 933c397078be24eba87cce63c3d49b507e62efb623a34f9349725461de719130 

[Download IVPN Client UI for Linux (DEB)](https://repo.ivpn.net/stable/pool/ivpn-ui_3.4.0_amd64.deb)  
SHA256: 7e50c58ed16c5817e79b253e7b198a76c4660218a1e236598a59a288eaaf89e3  
[Download IVPN Client UI for Linux (RPM)](https://repo.ivpn.net/stable/pool/ivpn-ui-3.4.0-1.x86_64.rpm)  
SHA256: cf95c4e07912aa03c7596d56b31d323664efbf44469cc9fee54771800d96d1db 

## Version 3.3.40 - 2021-09-14

[NEW] (Windows) Split Tunneling  

[Download IVPN Client for Windows](https://repo.ivpn.net/windows/bin/IVPN-Client-v3.3.40.exe)  
SHA256: 9875bc8ee2124464b66fa70555270865caf03c827e4323fdf6fb2a7a83589606  

## Version 3.3.30 - 2021-08-31

[NEW] Preparation for 2FA in CLI  
[IMPROVED] (Linux) The installation path changed from '/usr/local/bin' to '/usr/bin'  
[FIXED] Server selection order incorrect when sorted by country  
[FIXED] (Linux) Removed unnecessary paths from package which may lead to conflict with other software  

[Download IVPN Client for Windows](https://repo.ivpn.net/windows/bin/IVPN-Client-v3.3.30.exe)  
SHA256: 981bce29c543df2485687edcc9383e1fe5acc343cba0d8b8ea8beada8c57a3e6   

[Download IVPN Client for macOS](https://repo.ivpn.net/macos/bin/IVPN-3.3.30.dmg)  
SHA256: 7155967dda8f53580ab2d158fa57b447efe0c40a29f28b884bf33fc0f8fcb12d  

[Download IVPN Client for Linux (DEB)](https://repo.ivpn.net/stable/pool/ivpn_3.3.30_amd64.deb)  
SHA256: 89d20099b8e36b704106074c60a89ff189ff6e99e999a3ae748801b3ba76bd07   
[Download IVPN Client for Linux (RPM)](https://repo.ivpn.net/stable/pool/ivpn-3.3.30-1.x86_64.rpm)  
SHA256: 7b432c77c85bee2267bbbb218ee761b8c036208b14350476afa7179b133ad0a3 

[Download IVPN Client UI for Linux (DEB)](https://repo.ivpn.net/stable/pool/ivpn-ui_3.3.30_amd64.deb)  
SHA256: 229d70cfcb7bee5a7a888b5864797a5fec09cbd320f4d1a0c374cd30b17b2452  
[Download IVPN Client UI for Linux (RPM)](https://repo.ivpn.net/stable/pool/ivpn-ui-3.3.30-1.x86_64.rpm)  
SHA256: f7a77300bcc261af44e0d146970a89a4598d54be3161b2913516051d57f13a52  

## Version 3.3.20 - 2021-06-29

[NEW] IPv6 inside WireGuard tunnel  
[NEW] IPv6 connection info  
[NEW] New option in settings ‘Allow access to IVPN server when Firewall is enabled’  
[NEW] (Windows) Contrast tray icon (black or white; depends on Windows color theme)  
[FIXED] VPN was active after reboot when connected to Trusted WIFI  
[FIXED] Sometimes application was failing to connect to IVPN daemon  
[FIXED] (Windows) The daemon service was not starting when the 'Windows Events Logs' service is not running  
[FIXED] (macOS) WireGuard compatibility with old macOS versions  

[Download IVPN Client for Windows](https://repo.ivpn.net/windows/bin/IVPN-Client-v3.3.20.exe)  
SHA256: 02b312a0edf21765229c1e8f12e48bec2539b241e05afcda65b90c4f9730d950   

[Download IVPN Client for macOS](https://repo.ivpn.net/macos/bin/IVPN-3.3.20.dmg)  
SHA256: 14d4f51e2167a9c07e56d35de7632570168c69ed93beb4711128e5ddd04ca67f  

[Download IVPN Client for Linux (DEB)](https://repo.ivpn.net/stable/pool/ivpn_3.3.20_amd64.deb)  
SHA256: 9972a0b55e1383df67357d046db238b29c70b7865dcba959037da17b7439f20a   
[Download IVPN Client for Linux (RPM)](https://repo.ivpn.net/stable/pool/ivpn-3.3.20-1.x86_64.rpm)  
SHA256: 469d5b41b2092612cf82a9b269e4205ca4ebbcc651b5fbd196be8e138f005487 

[Download IVPN Client UI for Linux (DEB)](https://repo.ivpn.net/stable/pool/ivpn-ui_3.3.20_amd64.deb)  
SHA256: a9cd6f2e2e1c7f981d0788b0f6e381c56e8b44f44daad217b66b652d5eead947  
[Download IVPN Client UI for Linux (RPM)](https://repo.ivpn.net/stable/pool/ivpn-ui-3.3.20-1.x86_64.rpm)  
SHA256: 80e37b4c2c00fa89411e6bf403b72c60b66c09c2bd0ec0f0cdf264e76de00492  

## For old versions of IVPN for Desktop please refer to:

[Windows/macOS and Linux UI](https://github.com/ivpn/desktop-app-ui2/blob/master/CHANGELOG.md)  
[Linux (cli)](https://github.com/ivpn/desktop-app-cli/blob/master/CHANGELOG.md)
