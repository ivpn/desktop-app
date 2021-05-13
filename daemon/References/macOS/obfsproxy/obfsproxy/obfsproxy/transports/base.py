#!/usr/bin/python
# -*- coding: utf-8 -*-

import pyptlib.util

import obfsproxy.common.log as logging

import argparse

log = logging.get_obfslogger()

"""
This module contains BaseTransport, a pluggable transport skeleton class.
"""

def addrport(string):
    """
    Receive '<addr>:<port>' and return (<addr>,<port>).
    Used during argparse CLI parsing.
    """
    try:
        return pyptlib.util.parse_addr_spec(string, resolve=True)
    except ValueError, err:
        raise argparse.ArgumentTypeError(err)

class BaseTransport(object):
    """
    The BaseTransport class is a skeleton class for pluggable transports.
    It contains callbacks that your pluggable transports should
    override and customize.

    Attributes:
    circuit: Circuit object. This is set just before circuitConnected is called.
    """

    def __init__(self):
        """
        Initialize transport. This is called right after TCP connect.

        Subclass overrides should still call this via super().
        """
        self.name = "tran_%s" % hex(id(self))
        self.circuit = None

    @classmethod
    def setup(cls, pt_config):
        """
        Receive Pluggable Transport Config, perform setup task
        and save state in class attributes.
        Called at obfsproxy startup.

        Raise TransportSetupFailed if something goes wrong.
        """

    @classmethod
    def get_public_server_options(cls, transport_options):
        """
        By default all server transport options are passed to BridgeDB.
        If the transport server wishes to prevent some server
        transport options from being added to the BridgeDB then
        the transport may override this method and return a
        transport_options dict with the keys/values to be distributed.

        get_public_server_options receives the transport_options argument which
        is a dict of server transport options... for example:

        A torrc could specify multiple server transport options:

        ServerTransportPlugin bananaphone exec /usr/local/bin/obfsproxy --log-min-severity=debug --log-file=/var/log/tor/obfsproxy.log managed
        ServerTransportOptions bananaphone corpus=/opt/bananaphone-corpora/pg29468.txt encodingSpec=words,sha1,4 modelName=markov order=1

        But if the transport wishes to only pass the encodingSpec to
        the BridgeDB then get_public_server_options can be overridden like this:

        @classmethod
        def get_public_server_options(cls, transport_options):
            return dict(encodingSpec = transport_options['encodingSpec'])

        In this example the get_public_server_options receives the transport_options dict:
        {'corpus': '/opt/bananaphone-corpora/pg29468.txt', 'modelName': 'markov', 'order': '1', 'encodingSpec': 'words,sha1,4'}
        """
        return None

    def circuitConnected(self):
        """
        Our circuit was completed, and this is a good time to do your
        transport-specific handshake on its downstream side.
        """

    def circuitDestroyed(self, reason, side):
        """
        Our circuit was tore down.
        Both connections of the circuit are closed when this callback triggers.
        """

    def receivedDownstream(self, data):
        """
        Received 'data' in the downstream side of our circuit.
        'data' is an obfsproxy.network.buffer.Buffer.
        """

    def receivedUpstream(self, data):
        """
        Received 'data' in the upstream side of our circuit.
        'data' is an obfsproxy.network.buffer.Buffer.
        """

    def handle_socks_args(self, args):
        """
        'args' is a list of k=v strings that serve as configuration
        parameters to the pluggable transport.
        """

    @classmethod
    def register_external_mode_cli(cls, subparser):
        """
        Given an argparse ArgumentParser in 'subparser', register
        some default external-mode CLI arguments.

        Transports with more complex CLI are expected to override this
        function.
        """

        subparser.add_argument('mode', choices=['server', 'ext_server', 'client', 'socks'])
        subparser.add_argument('listen_addr', type=addrport)
        subparser.add_argument('--dest', type=addrport, help='Destination address')
        subparser.add_argument('--ext-cookie-file', type=str,
                               help='Filesystem path where the Extended ORPort authentication cookie is stored.')

    @classmethod
    def validate_external_mode_cli(cls, args):
        """
        Given the parsed CLI arguments in 'args', validate them and
        make sure they make sense. Return True if they are kosher,
        otherwise return False.

        Override for your own needs.
        """
        err = None

        # If we are not 'socks', we need to have a static destination
        # to send our data to.
        if (args.mode != 'socks') and (not args.dest):
            err = "'client' and 'server' modes need a destination address."

        elif (args.mode != 'ext_server') and args.ext_cookie_file:
            err = "No need for --ext-cookie-file if not an ext_server."

        elif (args.mode == 'ext_server') and (not args.ext_cookie_file):
            err = "You need to specify --ext-cookie-file as an ext_server."

        if not err: # We didn't encounter any errors during validation
            return True
        else: # Ugh, something failed.
            raise ValueError(err)

class PluggableTransportError(Exception): pass
class SOCKSArgsError(Exception): pass
class TransportSetupFailed(Exception): pass
