# Changelog

All notable changes to this project will be documented in this file.

## Version 2.12.10 - 2020-11-12
[IMPROVED] (macOS) Сompatibility with macOS Big Sur  
[FIXED] (macOS) Removed dependencies from WIFI libraries 

## Version 2.12.8 - 2020-11-09

[FIXED] (Linux) Firewall: Allow LAN functionality  
[FIXED] (Linux) Determine FastestServer when IVPN Firewall enabled 

## Version 2.12.7 - 2020-10-13

[NEW] Compatibility with the new IVPN GUI client  

## Version 2.12.5 - 2020-08-10

[FIXED] UI notification about the connection state  

## Version 2.12.4 - 2020-06-30

[IMPROVED] Minor improvements  

## Version 2.12.3 - 2020-06-05

[IMPROVED] User-defined extra configuration parameters for OpenVPN moved to separate file with access rights only for privileged account  
[FIXED] Random disconnections on waking-up from sleep  
[FIXED] (Linux) High CPU use with WireGuard connection  
[FIXED] (macOS) Always-on Firewall is blocking traffic on system boot  
[FIXED] (macOS) WireGuard connection error when a network interface not initialized  

## Version 2.12.2 - 2020-05-23

[IMPROVED] Overall stability  
[FIXED] Potential disconnection when network changes  

## Version 2.12.1 - 2020-05-21

[FIXED] Potential disconnection when network changes  

## Version 2.12.0 - 2020-05-14

[NEW] Command line interface for IVPN service  
[IMPROVED] Overall stability  

## Version 2.11.8 - 2020-04-05

[FIXED] macOS client start issue after the clean install

## Version 2.11.6 - 2020-03-27

[FIXED] Deadlock issue in 'ping' package

## Version 2.11.5 - 2020-03-26

[IMPROVED] Updated CA certificate for OpenVPN  
[FIXED] "Automatically change port" feature

## Version 2.11.4 - 2020-03-04

[FIXED] (Windows) Potential local privilege escalation vulnerability

## Version 2.11.3 - 2020-02-24

[FIXED] Pause feature for WireGuard  
[FIXED] Notify UI client that servers were updated

## Version 2.11.2 - 2020-02-20

[FIXED] (Windows) Unable to connect WireGuard if its service not uninstalled  
[FIXED] Firewall config changes from Always-On to On-Demand after upgrade  
[FIXED] Processing of users additional OpenVPN parameters

## Version 2.11.0 - 2020-01-28

[NEW] First version of IVPN Daemon written in Golang  
