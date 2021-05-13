from twisted.internet import reactor
from twisted.internet.protocol import Protocol, Factory

import obfsproxy.common.log as logging
import obfsproxy.common.heartbeat as heartbeat

import obfsproxy.network.buffer as obfs_buf
import obfsproxy.transports.base as base

log = logging.get_obfslogger()

"""
Networking subsystem:

A "Connection" is a bidirectional communications channel, usually
backed by a network socket. For example, the communication channel
between tor and obfsproxy is a 'connection'. In the code, it's
represented by a Twisted's twisted.internet.protocol.Protocol.

A 'Circuit' is a pair of connections, referred to as the 'upstream'
and 'downstream' connections. The upstream connection of a circuit
communicates in cleartext with the higher-level program that wishes to
make use of our obfuscation service. The downstream connection
communicates in an obfuscated fashion with the remote peer that the
higher-level client wishes to contact. In the code, it's represented
by the custom Circuit class.

The diagram below might help demonstrate the relationship between
connections and circuits:

                                   Downstream

       'Circuit C'      'Connection CD'   'Connection SD'     'Circuit S'
                     +-----------+          +-----------+
     Upstream    +---|Obfsproxy c|----------|Obfsproxy s|----+   Upstream
                 |   +-----------+    ^     +-----------+    |
 'Connection CU' |                    |                      | 'Connection SU'
           +------------+           Sent over       +--------------+
           | Tor Client |           the net         |  Tor Bridge  |
           +------------+                           +--------------+

In the above diagram, "Obfsproxy c" is the client-side obfsproxy, and
"Obfsproxy s" is the server-side obfsproxy. "Connection CU" is the
Client's Upstream connection, the communication channel between tor
and obfsproxy. "Connection CD" is the Client's Downstream connection,
the communication channel between obfsproxy and the remote peer. These
two connections form the client's circuit "Circuit C".

A 'listener' is a listening socket bound to a particular obfuscation
protocol, represented using Twisted's t.i.p.Factory. Connecting to a
listener creates one connection of a circuit, and causes this program
to initiate the other connection (possibly after receiving in-band
instructions about where to connect to). A listener is said to be a
'client' listener if connecting to it creates the upstream connection,
and a 'server' listener if connecting to it creates the downstream
connection.

There are two kinds of client listeners: a 'simple' client listener
always connects to the same remote peer every time it needs to
initiate a downstream connection; a 'socks' client listener can be
told to connect to an arbitrary remote peer using the SOCKS protocol.
"""

class Circuit(Protocol):
    """
    A Circuit holds a pair of connections. The upstream connection and
    the downstream. The circuit proxies data from one connection to
    the other.

    Attributes:
    transport: the pluggable transport we should use to
               obfuscate traffic on this circuit.

    downstream: the downstream connection
    upstream: the upstream connection
    """

    def __init__(self, transport):
        self.transport = transport # takes a transport
        self.downstream = None # takes a connection
        self.upstream = None # takes a connection

        self.closed = False # True if the circuit is closed.

        self.name = "circ_%s" % hex(id(self))

    def setDownstreamConnection(self, conn):
        """
        Set the downstream connection of a circuit.
        """

        log.debug("%s: Setting downstream connection (%s)." % (self.name, conn.name))
        assert(not self.downstream)
        self.downstream = conn

        if self.circuitIsReady():
            self.circuitCompleted(self.upstream)

    def setUpstreamConnection(self, conn):
        """
        Set the upstream connection of a circuit.
        """

        log.debug("%s: Setting upstream connection (%s)." % (self.name, conn.name))
        assert(not self.upstream)
        self.upstream = conn

        if self.circuitIsReady():
            self.circuitCompleted(self.downstream)

    def circuitIsReady(self):
        """
        Return True if the circuit is completed.
        """

        return self.downstream and self.upstream

    def circuitCompleted(self, conn_to_flush):
        """
        Circuit was just completed; that is, its endpoints are now
        connected. Do all the things we have to do now.
        """
        if self.closed:
            log.debug("%s: Completed circuit while closed. Ignoring.", self.name)
            return

        log.debug("%s: Circuit completed." % self.name)

        # Set us as the circuit of our pluggable transport instance.
        self.transport.circuit = self

        # Call the transport-specific circuitConnected method since
        # this is a good time to perform a handshake.
        self.transport.circuitConnected()

        # Do a dummy dataReceived on the initiating connection in case
        # it has any buffered data that must be flushed to the network.
        #
        # (We use callLater because we want to return back to the
        # event loop so that any messages we send in circuitConnected get sent
        # to the network immediately.)
        reactor.callLater(0.01, conn_to_flush.dataReceived, '')

    def dataReceived(self, data, conn):
        """
        We received 'data' on 'conn'. Pass the data to our transport,
        and then proxy it to the other side. # XXX 'data' is a buffer.

        Requires both downstream and upstream connections to be set.
        """
        if self.closed:
            log.debug("%s: Calling circuit's dataReceived while closed. Ignoring.", self.name)
            return

        assert(self.downstream and self.upstream)
        assert((conn is self.downstream) or (conn is self.upstream))

        try:
            if conn is self.downstream:
                log.debug("%s: downstream: Received %d bytes." % (self.name, len(data)))
                self.transport.receivedDownstream(data)
            else:
                log.debug("%s: upstream: Received %d bytes." % (self.name, len(data)))
                self.transport.receivedUpstream(data)
        except base.PluggableTransportError, err: # Our transport didn't like that data.
            log.info("%s: %s: Closing circuit." % (self.name, str(err)))
            self.close()

    def close(self, reason=None, side=None):
        """
        Tear down the circuit. The reason for the torn down circuit is given in
        'reason' and 'side' tells us where it happened: either upstream or
        downstream.
        """
        if self.closed:
            return # NOP if already closed

        log.debug("%s: Tearing down circuit." % self.name)

        self.closed = True

        if self.downstream:
            self.downstream.close()
        if self.upstream:
            self.upstream.close()

        self.transport.circuitDestroyed(reason, side)

class GenericProtocol(Protocol, object):
    """
    Generic obfsproxy connection. Contains useful methods and attributes.

    Attributes:
    circuit: The circuit object this connection belongs to.
    buffer: Buffer that holds data that can't be proxied right
            away. This can happen because the circuit is not yet
            complete, or because the pluggable transport needs more
            data before deciding what to do.
    """
    def __init__(self, circuit):
        self.circuit = circuit
        self.buffer = obfs_buf.Buffer()
        self.closed = False # True if connection is closed.

    def connectionLost(self, reason):
        log.debug("%s: Connection was lost (%s)." % (self.name, reason.getErrorMessage()))
        self.close()

    def connectionFailed(self, reason):
        log.debug("%s: Connection failed to connect (%s)." % (self.name, reason.getErrorMessage()))
        self.close()

    def write(self, buf):
        """
        Write 'buf' to the underlying transport.
        """
        if self.closed:
            log.debug("%s: Calling write() while connection is closed. Ignoring.", self.name)
            return

        log.debug("%s: Writing %d bytes." % (self.name, len(buf)))

        self.transport.write(buf)

    def close(self, also_close_circuit=True):
        """
        Close the connection.
        """
        if self.closed:
            return # NOP if already closed

        log.debug("%s: Closing connection." % self.name)

        self.closed = True

        self.transport.loseConnection()
        if also_close_circuit:
            self.circuit.close()


class StaticDestinationProtocol(GenericProtocol):
    """
    Represents a connection to a static destination (as opposed to a
    SOCKS connection).

    Attributes:
    mode: 'server' or 'client'
    circuit: The circuit this connection belongs to.

    buffer: Buffer that holds data that can't be proxied right
            away. This can happen because the circuit is not yet
            complete, or because the pluggable transport needs more
            data before deciding what to do.
    """

    def __init__(self, circuit, mode, peer_addr):
        self.mode = mode
        self.peer_addr = peer_addr
        self.name = "conn_%s" % hex(id(self))

        GenericProtocol.__init__(self, circuit)

    def connectionMade(self):
        """
        Callback for when a connection is successfully established.

        Find the connection's direction in the circuit, and register
        it in our circuit.
        """

        # Find the connection's direction and register it in the circuit.
        if self.mode == 'client' and not self.circuit.upstream:
            log.debug("%s: connectionMade (client): " \
                      "Setting it as upstream on our circuit." % self.name)

            self.circuit.setUpstreamConnection(self)
        elif self.mode == 'client':
            log.debug("%s: connectionMade (client): " \
                      "Setting it as downstream on our circuit." % self.name)

            self.circuit.setDownstreamConnection(self)
        elif self.mode == 'server' and not self.circuit.downstream:
            log.debug("%s: connectionMade (server): " \
                      "Setting it as downstream on our circuit." % self.name)

            # Gather some statistics for our heartbeat.
            heartbeat.heartbeat.register_connection(self.peer_addr.host)

            self.circuit.setDownstreamConnection(self)
        elif self.mode == 'server':
            log.debug("%s: connectionMade (server): " \
                      "Setting it as upstream on our circuit." % self.name)

            self.circuit.setUpstreamConnection(self)

    def dataReceived(self, data):
        """
        We received some data from the network. See if we have a
        complete circuit, and pass the data to it they get proxied.

        XXX: Can also be called with empty 'data' because of
        Circuit.setDownstreamConnection(). Document or split function.
        """
        if self.closed:
            log.debug("%s: dataReceived called while closed. Ignoring.", self.name)
            return

        if (not self.buffer) and (not data):
            log.debug("%s: dataReceived called without a reason.", self.name)
            return

        # Add the received data to the buffer.
        self.buffer.write(data)

        # Circuit is not fully connected yet, nothing to do here.
        if not self.circuit.circuitIsReady():
            log.debug("%s: Incomplete circuit; cached %d bytes." % (self.name, len(data)))
            return

        self.circuit.dataReceived(self.buffer, self)

class StaticDestinationClientFactory(Factory):
    """
    Created when our listener receives a client connection. Makes the
    connection that connects to the other end of the circuit.
    """

    def __init__(self, circuit, mode):
        self.circuit = circuit
        self.mode = mode

        self.name = "fact_c_%s" % hex(id(self))

    def buildProtocol(self, addr):
        return StaticDestinationProtocol(self.circuit, self.mode, addr)

    def startedConnecting(self, connector):
        log.debug("%s: Client factory started connecting." % self.name)

    def clientConnectionLost(self, connector, reason):
        pass # connectionLost event is handled on the Protocol.

    def clientConnectionFailed(self, connector, reason):
        log.debug("%s: Connection failed (%s)." % (self.name, reason.getErrorMessage()))
        self.circuit.close()

class StaticDestinationServerFactory(Factory):
    """
    Represents a listener. Upon receiving a connection, it creates a
    circuit and tries to establish the other side of the circuit. It
    then listens for data to obfuscate and proxy.

    Attributes:

    remote_host: The IP/DNS information of the host on the other side
                 of the circuit.
    remote_port: The TCP port fo the host on the other side of the circuit.
    mode: 'server' or 'client'
    transport: the pluggable transport we should use to
               obfuscate traffic on this connection.
    pt_config: an object containing config options for the transport.
    """
    def __init__(self, remote_addrport, mode, transport_class, pt_config):
        self.remote_host = remote_addrport[0]
        self.remote_port = int(remote_addrport[1])
        self.mode = mode
        self.transport_class = transport_class
        self.pt_config = pt_config

        self.name = "fact_s_%s" % hex(id(self))

        assert(self.mode == 'client' or self.mode == 'server')

    def startFactory(self):
        log.debug("%s: Starting up static destination server factory." % self.name)

    def buildProtocol(self, addr):
        log.debug("%s: New connection from %s:%d." % (self.name, log.safe_addr_str(addr.host), addr.port))

        circuit = Circuit(self.transport_class())

        # XXX instantiates a new factory for each client
        clientFactory = StaticDestinationClientFactory(circuit, self.mode)

        if self.pt_config.proxy:
            create_proxy_client(self.remote_host, self.remote_port,
                                self.pt_config.proxy,
                                clientFactory)
        else:
            reactor.connectTCP(self.remote_host, self.remote_port, clientFactory)

        return StaticDestinationProtocol(circuit, self.mode, addr)

def create_proxy_client(host, port, proxy_spec, instance):
    """
    host:
    the host of the final destination
    port:
    the port number of the final destination
    proxy_spec:
    the address of the proxy server as a urlparse.SplitResult
    instance:
    is the instance to be associated with the endpoint

    Returns a deferred that will fire when the connection to the SOCKS server has been established.
    """

    # Inline import so that txsocksx is an optional dependency.
    from twisted.internet.endpoints import HostnameEndpoint
    from txsocksx.client import SOCKS4ClientEndpoint, SOCKS5ClientEndpoint
    from obfsproxy.network.http import HTTPConnectClientEndpoint

    TCPPoint = HostnameEndpoint(reactor, proxy_spec.hostname, proxy_spec.port)
    username = proxy_spec.username
    password = proxy_spec.password

    # Do some logging
    log.debug("Connecting via %s proxy %s:%d",
              proxy_spec.scheme, log.safe_addr_str(proxy_spec.hostname), proxy_spec.port)
    if username or password:
        log.debug("Using %s:%s as the proxy credentials",
                  log.safe_addr_str(username), log.safe_addr_str(password))

    if proxy_spec.scheme in ["socks4a", "socks5"]:
        if proxy_spec.scheme == "socks4a":
            if username:
                assert(password == None)
                SOCKSPoint = SOCKS4ClientEndpoint(host, port, TCPPoint, user=username)
            else:
                SOCKSPoint = SOCKS4ClientEndpoint(host, port, TCPPoint)
        elif proxy_spec.scheme == "socks5":
            if username and password:
                SOCKSPoint = SOCKS5ClientEndpoint(host, port, TCPPoint,
                                                  methods={'login': (username, password)})
            else:
                assert(username == None and password == None)
                SOCKSPoint = SOCKS5ClientEndpoint(host, port, TCPPoint)
        d = SOCKSPoint.connect(instance)
        return d
    elif proxy_spec.scheme == "http":
        if username and password:
            HTTPPoint = HTTPConnectClientEndpoint(host, port, TCPPoint,
                                                  username, password)
        else:
            assert(username == None and password == None)
            HTTPPoint = HTTPConnectClientEndpoint(host, port, TCPPoint)
        d = HTTPPoint.connect(instance)
        return d
    else:
        # Should *NEVER* happen
        raise RuntimeError("Invalid proxy scheme %s" % proxy_spec.scheme)

def ensure_outgoing_proxy_dependencies():
    """Make sure that we have the necessary dependencies to connect to
    outgoing HTTP/SOCKS proxies.

    Raises OutgoingProxyDepsFailure in case of error.
    """

    # We can't connect to outgoing proxies without txsocksx.
    try:
        import txsocksx
    except ImportError:
        raise OutgoingProxyDepsFailure("We don't have txsocksx. Can't do proxy. Please install txsocksx.")

    # We also need a recent version of twisted ( >= twisted-13.2.0)
    import twisted
    from twisted.python import versions
    if twisted.version < versions.Version('twisted', 13, 2, 0):
        raise OutgoingProxyDepsFailure("Outdated version of twisted (%s). Please upgrade to >= twisted-13.2.0" % twisted.version.short())

class OutgoingProxyDepsFailure(Exception): pass
