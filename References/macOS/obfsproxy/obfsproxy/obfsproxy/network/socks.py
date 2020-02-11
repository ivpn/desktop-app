import csv

from twisted.internet import reactor, protocol

import obfsproxy.common.log as logging
import obfsproxy.network.network as network
import obfsproxy.network.socks5 as socks5
import obfsproxy.transports.base as base


log = logging.get_obfslogger()


def _split_socks_args(args_str):
    """
    Given a string containing the SOCKS arguments (delimited by
    semicolons, and with semicolons and backslashes escaped), parse it
    and return a list of the unescaped SOCKS arguments.
    """
    return csv.reader([args_str], delimiter=';', escapechar='\\').next()


class OBFSSOCKSv5Outgoing(socks5.SOCKSv5Outgoing, network.GenericProtocol):
    """
    Represents a downstream connection from the SOCKS server to the
    destination.

    It subclasses socks5.SOCKSv5Outgoing, so that data can be passed to the
    pluggable transport before proxying.

    Attributes:
    circuit: The circuit this connection belongs to.
    buffer: Buffer that holds data that can't be proxied right
            away. This can happen because the circuit is not yet
            complete, or because the pluggable transport needs more
            data before deciding what to do.
    """

    name = None

    def __init__(self, socksProtocol):
        """
        Constructor.

        'socksProtocol' is a 'SOCKSv5Protocol' object.
        """
        self.name = "socks_down_%s" % hex(id(self))
        self.socks = socksProtocol

        network.GenericProtocol.__init__(self, socksProtocol.circuit)
        return super(OBFSSOCKSv5Outgoing, self).__init__(socksProtocol)

    def connectionMade(self):
        self.socks.set_up_circuit(self)

        # XXX: The transport should be doing this after handshaking since it
        # calls, self.socks.sendReply(), when this changes to defer sending the
        # reply back set self.socks.otherConn here.
        super(OBFSSOCKSv5Outgoing, self).connectionMade()

    def dataReceived(self, data):
        log.debug("%s: Recived %d bytes." % (self.name, len(data)))

        assert self.circuit.circuitIsReady()
        self.buffer.write(data)
        self.circuit.dataReceived(self.buffer, self)

class OBFSSOCKSv5OutgoingFactory(protocol.Factory):
    """
    A OBFSSOCKSv5OutgoingFactory, used only when connecting via a proxy
    """

    def __init__(self, socksProtocol):
        self.socks = socksProtocol

    def buildProtocol(self, addr):
        return OBFSSOCKSv5Outgoing(self.socks)

    def clientConnectionFailed(self, connector, reason):
        self.socks.transport.loseConnection()

    def clientConnectionLost(self, connector, reason):
        self.socks.transport.loseConnection()

class OBFSSOCKSv5Protocol(socks5.SOCKSv5Protocol, network.GenericProtocol):
    """
    Represents an upstream connection from a SOCKS client to our SOCKS
    server.

    It overrides socks5.SOCKSv5Protocol because py-obfsproxy's connections need
    to have a circuit and obfuscate traffic before proxying it.
    """

    def __init__(self, circuit, pt_config):
        self.name = "socks_up_%s" % hex(id(self))
        self.pt_config = pt_config

        network.GenericProtocol.__init__(self, circuit)
        socks5.SOCKSv5Protocol.__init__(self)

    def connectionLost(self, reason):
        network.GenericProtocol.connectionLost(self, reason)

    def processEstablishedData(self, data):
        assert self.circuit.circuitIsReady()
        self.buffer.write(data)
        self.circuit.dataReceived(self.buffer, self)

    def processRfc1929Auth(self, uname, passwd):
        """
        Handle the Pluggable Transport variant of RFC1929 Username/Password
        authentication.
        """

        # The Tor PT spec jams the per session arguments into the UNAME/PASSWD
        # fields, and uses this to pass arguments to the pluggable transport.

        # Per the RFC, it's not possible to have 0 length passwords, so tor sets
        # the length to 1 and the first byte to NUL when passwd doesn't actually
        # contain data.  Recombine the two fields if appropriate.
        args = uname
        if len(passwd) > 1 or ord(passwd[0]) != 0:
            args += passwd

        # Arguments are a CSV string with Key=Value pairs.  The transport is
        # responsible for dealing with the K=V format, but the SOCKS code is
        # currently expected to de-CSV the args.
        #
        # XXX: This really should also handle converting the K=V pairs into a
        # dict.
        try:
            split_args = _split_socks_args(args)
        except csvError, err:
            log.warning("split_socks_args failed (%s)" % str(err))
            return False

        # Pass the split up list to the transport.
        try:
            self.circuit.transport.handle_socks_args(split_args)
        except base.SOCKSArgsError:
            # Transports should log the issue themselves
            return False

        return True

    def connectClass(self, addr, port, klass, *args):
        """
        Instantiate the outgoing connection.

        This is overriden so that our sub-classed SOCKSv5Outgoing gets created,
        and a proxy is optionally used for the outgoing connection.
        """

        if self.pt_config.proxy:
            instance = OBFSSOCKSv5OutgoingFactory(self)
            return network.create_proxy_client(addr, port, self.pt_config.proxy, instance)
        else:
            return protocol.ClientCreator(reactor, OBFSSOCKSv5Outgoing, self).connectTCP(addr, port)

    def set_up_circuit(self, otherConn):
        self.circuit.setDownstreamConnection(otherConn)
        self.circuit.setUpstreamConnection(self)

class OBFSSOCKSv5Factory(protocol.Factory):
    """
    A SOCKSv5 factory.
    """

    def __init__(self, transport_class, pt_config):
        # XXX self.logging = log
        self.transport_class = transport_class
        self.pt_config = pt_config

        self.name = "socks_fact_%s" % hex(id(self))

    def startFactory(self):
        log.debug("%s: Starting up SOCKS server factory." % self.name)

    def buildProtocol(self, addr):
        log.debug("%s: New connection." % self.name)

        circuit = network.Circuit(self.transport_class())

        return OBFSSOCKSv5Protocol(circuit, self.pt_config)
