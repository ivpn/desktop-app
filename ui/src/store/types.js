//
//  UI for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
//
//  This file is part of the UI for IVPN Client Desktop.
//
//  The UI for IVPN Client Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The UI for IVPN Client Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the UI for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
//

export const DaemonConnectionType = Object.freeze({
  NotConnected: 0,
  Connecting: 1,
  Connected: 2,
});

export const VpnTypeEnum = Object.freeze({ OpenVPN: 0, WireGuard: 1 });
export const PauseStateEnum = Object.freeze({
  Resumed: 0,
  Pausing: 1,
  Paused: 2,
  Resuming: 3,
});

export const DnsEncryption = Object.freeze({
  None: 0,
  DnsOverTls: 1,
  DnsOverHttps: 2,
});

export const VpnStateEnum = Object.freeze({
  DISCONNECTED: 0,
  CONNECTING: 1, // OpenVPN's initial state.
  WAIT: 2, // (Client only) Waiting for initial response from server.
  AUTH: 3, // (Client only) Authenticating with server.
  GETCONFIG: 4, // (Client only) Downloading configuration options from server.
  ASSIGNIP: 5, // Assigning IP address to virtual network interface.
  ADDROUTES: 6, // Adding routes to system.
  CONNECTED: 7, // Initialization Sequence Completed.
  RECONNECTING: 8, // A restart has occurred.
  TCP_CONNECT: 9, // TCP_CONNECT
  EXITING: 10, // A graceful exit is in progress.
  DISCONNECTING: 11,
});

export const PingQuality = Object.freeze({ Good: 0, Moderate: 1, Bad: 2 });

export const PortTypeEnum = Object.freeze({ UDP: 0, TCP: 1 });

export const ObfsproxyVerEnum = Object.freeze({ obfs3: 3, obfs4: 4 });
export const Obfs4IatEnum = Object.freeze({
  IAT0: 0,
  IAT1: 1,
  IAT2: 2,
});

export const ServersSortTypeEnum = Object.freeze({
  City: 0,
  Country: 1,
  Latency: 2,
  Proximity: 3,
});

export const ColorTheme = Object.freeze({
  system: "system",
  light: "light",
  dark: "dark",
});

export const AppUpdateStage = Object.freeze({
  NoStatus: "No update status",
  CheckingForUpdates: "Checking for app updates...",
  CheckingFinished: "Checking for app updates finished",
  CancelledDownload: "Download cancelled",
  Downloading: "Downloading ...",
  CheckingSignature: "Checking signature ...",
  ReadyToInstall: "Ready to install",
  Installing: "Installing...",
  Error: "Error",
});

export function NormalizedConfigPortObject(portFromServersConfig) {
  if (
    !portFromServersConfig ||
    !portFromServersConfig.port ||
    portFromServersConfig.type == null ||
    portFromServersConfig.type == undefined
  )
    return null;

  const p = parseInt(portFromServersConfig.port, 10);
  if (isNaN(p)) return null;

  return {
    port: p,
    type:
      portFromServersConfig.type === PortTypeEnum.TCP || // the type can be already converted value
      (typeof portFromServersConfig.type === "string" &&
        portFromServersConfig.type.trim().toUpperCase() == "TCP")
        ? PortTypeEnum.TCP
        : PortTypeEnum.UDP,
  };
}

export function NormalizedConfigPortRangeObject(portFromServersConfig) {
  if (!portFromServersConfig) return null;

  const range = portFromServersConfig.range;
  if (
    !range ||
    !range.min ||
    !range.max ||
    portFromServersConfig.type == null ||
    portFromServersConfig.type == undefined
  )
    return null;

  const r = {
    min: parseInt(range.min, 10),
    max: parseInt(range.max, 10),
  };

  if (isNaN(r.min) || isNaN(r.max)) return null;
  if (r.min <= 0 || r.min > range.max) return null;

  return {
    range: r,
    type:
      portFromServersConfig.type === PortTypeEnum.TCP || // the type can be already converted value
      (typeof portFromServersConfig.type === "string" &&
        portFromServersConfig.type.trim().toUpperCase() == "TCP")
        ? PortTypeEnum.TCP
        : PortTypeEnum.UDP,
  };
}
