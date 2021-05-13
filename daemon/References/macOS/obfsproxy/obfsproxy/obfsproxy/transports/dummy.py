#!/usr/bin/python
# -*- coding: utf-8 -*-

""" This module contains an implementation of the 'dummy' transport. """

from obfsproxy.transports.base import BaseTransport


class DummyTransport(BaseTransport):
    """
    Implements the dummy protocol. A protocol that simply proxies data
    without obfuscating them.
    """

    def __init__(self):
        """
        If you override __init__, you ought to call the super method too.
        """

        super(DummyTransport, self).__init__()

    def receivedDownstream(self, data):
        """
        Got data from downstream; relay them upstream.
        """

        self.circuit.upstream.write(data.read())

    def receivedUpstream(self, data):
        """
        Got data from upstream; relay them downstream.
        """

        self.circuit.downstream.write(data.read())

class DummyClient(DummyTransport):

    """
    DummyClient is a client for the 'dummy' protocol.
    Since this protocol is so simple, the client and the server are identical and both just trivially subclass DummyTransport.
    """

    pass


class DummyServer(DummyTransport):

    """
    DummyServer is a server for the 'dummy' protocol.
    Since this protocol is so simple, the client and the server are identical and both just trivially subclass DummyTransport.
    """

    pass


