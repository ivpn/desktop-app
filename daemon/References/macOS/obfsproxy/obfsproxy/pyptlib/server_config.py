#!/usr/bin/python
# -*- coding: utf-8 -*-

"""
Low-level parts of pyptlib that are only useful to servers.
"""

import pyptlib.config as config
import pyptlib.util as util

from pyptlib.config import env_has_k, get_env, SUPPORTED_TRANSPORT_VERSIONS

def get_transport_options_impl(string):
    """
    Parse transport options.
    :param str optstring: Example input: 'scramblesuit:k=v;scramblesuit:k2=v2;obs3fs:k=v'
    :returns: {'obfs3': {'k':'v'}, 'scramblesuit': {'k2' : 'v2', 'k' : 'v'} }
    """
    transport_args = {}

    params = string.split(';')
    for param in params:
        try:
            (name, kv_string) = param.split(':')
        except ValueError:
            raise ValueError("Invalid options string (%s)" % param)

        if name not in transport_args:
            transport_args[name] = {}

        try:
            (key, value) = kv_string.split('=')
        except ValueError:
            raise ValueError("Not a k=v value (%s)" % kv_string)

        transport_args[name][key] = value

    return transport_args

class ServerConfig(config.Config):
    """
    A client-side pyptlib configuration.

    :var tuple ORPort: (ip,port) pointing to Tor's ORPort.
    :var tuple extendedORPort: (ip,port) pointing to Tor's Extended ORPort. None if Extended ORPort is not supported.
    :var dict serverBindAddr: A dictionary {<transport> : [<addr>, <port>]}, where <transport> is the name of the transport that must be spawned, and [<addr>, <port>] is a list containing the location where that transport should bind. The dictionary can be empty.
    :var string authCookieFile: String representing the filesystem path where the Extended ORPort Authentication cookie is stored. None if Extended ORPort authentication is not supported.
    :var dict serverTransportOptions: Dictionary containing user-provided parameters that must be passed to the pluggable transports.
        Example: {'obfs3': {'k':'v'}, 'scramblesuit': {'k2' : 'v2', 'k' : 'v'} }
    """

    @classmethod
    def fromEnv(cls):
        """
        Build a ServerConfig from environment variables.

        :raises: :class:`pyptlib.config.EnvError` if environment was incomplete or corrupted.
        """

        # TOR_PT_EXTENDED_SERVER_PORT is set and empty if Tor does not support
        # the Extended ORPort.
        def empty_or_valid_addr(k, v):
            v = env_has_k(k, v)
            if v == '': return None
            return util.parse_addr_spec(v)

        extendedORPort = get_env('TOR_PT_EXTENDED_SERVER_PORT', empty_or_valid_addr)

        # Check that either both Extended ORPort and the Extended
        # ORPort Authentication Cookie are present, or neither.
        if extendedORPort:
            def get_authcookie(_, v):
                if v is None: raise ValueError("Extended ORPort address provided, but no cookie file.")
                return v
        else:
            def get_authcookie(_, v):
                if v is not None: raise ValueError("Extended ORPort Authentication cookie file provided, but no Extended ORPort address.")
                return v
        authCookieFile = get_env('TOR_PT_AUTH_COOKIE_FILE', get_authcookie)

        # Get ORPort.
        ORPort = get_env('TOR_PT_ORPORT', empty_or_valid_addr)

        # Get bind addresses.
        def get_server_bindaddr(k, bindaddrs):
            serverBindAddr = {}
            bindaddrs = env_has_k(k, bindaddrs).split(',')
            for bindaddr in bindaddrs:
                (transport_name, addrport) = bindaddr.split('-')
                (addr, port) = util.parse_addr_spec(addrport)
                serverBindAddr[transport_name] = (addr, port)
            return serverBindAddr
        serverBindAddr = get_env('TOR_PT_SERVER_BINDADDR', get_server_bindaddr)

        # Get transports.
        def get_transports(k, transports):
            transports = env_has_k(k, transports).split(',')
            t = sorted(transports)
            b = sorted(serverBindAddr.keys())
            if t != b:
                raise ValueError("Can't match transports with bind addresses (%s, %s)" % (t, b))
            return transports
        transports = get_env('TOR_PT_SERVER_TRANSPORTS', get_transports)

        def get_transport_options(k, v):
            if v is None:
                return None
            serverTransportOptions = env_has_k(k, v)
            return get_transport_options_impl(serverTransportOptions)
        transport_options = get_env('TOR_PT_SERVER_TRANSPORT_OPTIONS', get_transport_options)

        return cls(
            stateLocation = get_env('TOR_PT_STATE_LOCATION'),
            managedTransportVer = get_env('TOR_PT_MANAGED_TRANSPORT_VER').split(','),
            transports = transports,
            serverBindAddr = serverBindAddr,
            ORPort = ORPort,
            extendedORPort = extendedORPort,
            authCookieFile = authCookieFile,
            serverTransportOptions = transport_options
            )

    def __init__(self, stateLocation,
                 managedTransportVer=None,
                 transports=None,
                 serverBindAddr=None,
                 ORPort=None,
                 extendedORPort=None,
                 authCookieFile=None,
                 serverTransportOptions=None):
        config.Config.__init__(self, stateLocation,
            managedTransportVer or SUPPORTED_TRANSPORT_VERSIONS,
            transports or [])
        self.serverBindAddr = serverBindAddr or {}
        self.ORPort = ORPort
        self.extendedORPort = extendedORPort
        self.authCookieFile = authCookieFile
        self.serverTransportOptions = serverTransportOptions

    def getExtendedORPort(self):
        """
        :returns: :attr:`pyptlib.server_config.ServerConfig.extendedORPort`
        """
        return self.extendedORPort

    def getORPort(self):
        """
        :returns: :attr:`pyptlib.server_config.ServerConfig.ORPort`
        """
        return self.ORPort

    def getAuthCookieFile(self):
        """
        :returns: :attr:`pyptlib.server_config.ServerConfig.authCookieFile`
        """
        return self.authCookieFile

    def getServerTransportOptions(self):
        """
        :returns: :attr:`pyptlib.server_config.ServerConfig.serverTransportOptions`
        """
        return self.serverTransportOptions
