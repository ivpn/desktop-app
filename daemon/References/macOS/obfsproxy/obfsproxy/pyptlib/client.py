#!/usr/bin/python
# -*- coding: utf-8 -*-

"""
Public client-side pyptlib API.
"""

from pyptlib.core import TransportPlugin
from pyptlib.client_config import ClientConfig


class ClientTransportPlugin(TransportPlugin):
    """
    Runtime process for a client TransportPlugin.
    """
    configType = ClientConfig
    methodName = 'CMETHOD'
    reportedProxy = False

    def reportMethodSuccess(self, name, protocol, addrport, args=None, optArgs=None):
        """
        Write a message to stdout announcing that a transport was
        successfully launched.

        :param str name: Name of transport.
        :param str protocol: Name of protocol to communicate using.
        :param tuple addrport: (addr,port) where this transport is listening for connections.
        :param str args: ARGS field for this transport.
        :param str optArgs: OPT-ARGS field for this transport.
        """

        methodLine = 'CMETHOD %s %s %s:%s' % (name, protocol,
                addrport[0], addrport[1])
        if args and len(args) > 0:
            methodLine = methodLine + ' ARGS=' + args.join(',')
        if optArgs and len(optArgs) > 0:
            methodLine = methodLine + ' OPT-ARGS=' + args.join(',')
        self.emit(methodLine)

    def reportProxySuccess(self):
        """
        Write a message to stdout announcing that the specified proxy will be
        used.
        """

        if not self.config.proxy:
            raise RuntimeError("reportProxySuccess() when no proxy specified")
        elif self.reportedProxy:
            raise RuntimeError("reportProxySuccess() after status already reported")
        else:
            self.reportedProxy = True
            self.emit("PROXY DONE")

    def reportProxyError(self, msg=None):
        """
        Write a message to stdout announcing that the specified proxy can not be
        used.
        """

        if not self.config.proxy:
            raise RuntimeError("reportProxyError() when no proxy specified")
        elif self.reportedProxy:
            raise RuntimeError("reportProxyError() after status already reported")
        else:
            self.reportedProxy = True
            proxyLine = 'PROXY-ERROR'
            if msg and len(msg) > 0:
                proxyLine += ' ' + msg
            self.emit(proxyLine)

def init(supported_transports):
    """DEPRECATED. Use ClientTransportPlugin().init() instead."""
    client = ClientTransportPlugin()

    client.init(supported_transports)
    retval = {}
    retval['state_loc'] = client.config.getStateLocation()
    retval['transports'] = client.getTransports()

    return retval

def reportSuccess(name, socksVersion, addrport, args=None, optArgs=None):
    """DEPRECATED. Use ClientTransportPlugin().reportMethodSuccess() instead."""
    config = ClientTransportPlugin()
    config.reportMethodSuccess(name, "socks%s" % socksVersion, addrport, args, optArgs)

def reportFailure(name, message):
    """DEPRECATED. Use ClientTransportPlugin().reportMethodError() instead."""
    config = ClientTransportPlugin()
    config.reportMethodError(name, message)

def reportEnd():
    """DEPRECATED. Use ClientTransportPlugin().reportMethodsEnd() instead."""
    config = ClientTransportPlugin()
    config.reportMethodsEnd()
