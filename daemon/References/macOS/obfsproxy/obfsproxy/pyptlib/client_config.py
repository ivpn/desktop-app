#!/usr/bin/python
# -*- coding: utf-8 -*-

"""
Low-level parts of pyptlib that are only useful to clients.
"""

from pyptlib.config import Config, ProxyError, get_env
from pyptlib import util
from urlparse import urlsplit

SUPPORTED_PROXY_SCHEMES = ['http', 'socks4a', 'socks5']

class ClientConfig(Config):
    """
    A client-side pyptlib configuration.

    :var urlparse.SplitResult proxy: The proxy that should be used for outgoing connections.  None if no proxy is required.
    """

    @classmethod
    def fromEnv(cls):
        """
        Build a ClientConfig from environment variables.

        :raises: :class:`pyptlib.config.EnvError` if environment was incomplete or corrupted.
        :raises: :class:`pyptlib.config.ProxyError` if proxy was incomplete or corrupted.
        """

        # TOR_PT_PROXY is either totally missing or is a valid URI specifying
        # the proxy that pluggable transports should use.
        def missing_or_valid_proxy_uri(k, v):
            if v is None: return v
            return parseProxyURI(v)

        return cls(
            stateLocation = get_env('TOR_PT_STATE_LOCATION'),
            managedTransportVer = get_env('TOR_PT_MANAGED_TRANSPORT_VER').split(','),
            transports = get_env('TOR_PT_CLIENT_TRANSPORTS').split(','),
            proxy = get_env('TOR_PT_PROXY', missing_or_valid_proxy_uri)
            )

    def __init__(self,
                 stateLocation,
                 managedTransportVer=None,
                 transports=None,
                 proxy=None):
        Config.__init__(self, stateLocation, managedTransportVer, transports)
        self.proxy = proxy

    def getProxy(self):
        """
        Get the proxy that should be used for outgoing connections if any.

        The returned urlparse.SplitResult has the following attributes:
         * scheme - one of 'socks4a', 'socks5', or 'https'
         * hostname - Hostname of the proxy
         * port - TCP port of the proxy
         * username - Username to use for proxy authentication (optional)
         * password - Password to use for proxy authentication (optional)

        :returns: :attr:`pyptlib.client_config.ClientConfig.proxy`

        """
        return self.proxy

def parseProxyURI(uri_str):
    try:
        uri = urlsplit(uri_str, allow_fragments=False)
    except Exception, e:
        raise ProxyError("Error parsing proxy URI (%s)" % uri_str)

    if not uri.scheme in SUPPORTED_PROXY_SCHEMES:
        raise ProxyError("Invalid scheme (%s)" % uri.scheme)
    if uri.scheme == 'socks4a' and uri.password:
        raise ProxyError("Proxy URI specified SOCKS4a and a password")
    elif uri.scheme == 'socks5':
        if uri.username and not uri.password:
            raise ProxyError("Proxy URI specified SOCKS5, a username and no password")
        if uri.password and not uri.username:
            raise ProxyError("Proxy URI specified SOCKS5, a password and no username")
        if uri.username and len(uri.username) > 255:
            raise ProxyError("Proxy URI specified an oversized username")
        if uri.password and len(uri.password) > 255:
            raise ProxyError("Proxy URI specified an oversized password")
    if uri.netloc == '':
        raise ProxyError("Proxy URI is missing a netloc (%s)" % uri.netloc)
    if not uri.hostname:
        raise ProxyError("Proxy URI is missing a hostname")
    if not uri.port:
        raise ProxyError("Proxy URI is missing a port")
    if uri.path != '':
        raise ProxyError("Proxy URI has a path when none expected (%s)" %
                         uri.path)
    if uri.query != '':
        raise ProxyError("Proxy URI has query when none expected (%s)" %
                         uri.query)
    try:
        addr_str = uri.hostname + ":" + str(uri.port)
        host, port = util.parse_addr_spec(addr_str)
    except:
        raise ProxyError("Proxy URI has invalid netloc (%s)" %
                         uri.netloc)
    return uri
