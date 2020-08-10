export const VpnTypeEnum = Object.freeze({ OpenVPN: 0, WireGuard: 1 });
export const PauseStateEnum = Object.freeze({
  Resumed: 0,
  Pausing: 1,
  Paused: 2,
  Resuming: 3
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
  DISCONNECTING: 11
});

export const PingQuality = Object.freeze({ Good: 0, Moderate: 1, Bad: 2 });

export const PortTypeEnum = Object.freeze({ UDP: 0, TCP: 1 });
export const Ports = Object.freeze({
  OpenVPN: [
    { port: 2049, type: PortTypeEnum.UDP },
    { port: 2050, type: PortTypeEnum.UDP },
    { port: 53, type: PortTypeEnum.UDP },
    { port: 1194, type: PortTypeEnum.UDP },
    { port: 443, type: PortTypeEnum.TCP },
    { port: 1443, type: PortTypeEnum.TCP },
    { port: 80, type: PortTypeEnum.TCP }
  ],
  WireGuard: [
    { port: 2049, type: PortTypeEnum.UDP },
    { port: 2050, type: PortTypeEnum.UDP },
    { port: 53, type: PortTypeEnum.UDP },
    { port: 1194, type: PortTypeEnum.UDP },
    { port: 30587, type: PortTypeEnum.UDP },
    { port: 41893, type: PortTypeEnum.UDP },
    { port: 48574, type: PortTypeEnum.UDP },
    { port: 58237, type: PortTypeEnum.UDP }
  ]
});
