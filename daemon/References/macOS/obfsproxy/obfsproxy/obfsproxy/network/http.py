from base64 import b64encode
from twisted.internet.error import ConnectError
from twisted.internet.interfaces import IStreamClientEndpoint
from twisted.internet.protocol import ClientFactory
from twisted.internet.defer import Deferred
from twisted.web.http import HTTPClient
from zope.interface import implementer

import obfsproxy.common.log as logging

"""
HTTP CONNECT Client:

Next up on the list of things one would expect Twisted to provide, but does not
is an endpoint for outgoing connections through a HTTP CONNECT proxy.

Limitations:
 * Only Basic Authentication is supported (RFC2617).
"""

log = logging.get_obfslogger()

# Create the body of the RFC2617 Basic Authentication 'Authorization' header.
def _makeBasicAuth(username, password):
    if username and password:
        return "Basic " + b64encode(username + ':' + password)
    elif username or password:
        raise ValueError("expecting both a username *and* password")
    else:
        return None

class HTTPConnectClient(HTTPClient):
    deferred = None
    host = None
    port = None
    proxy_addr = None
    auth = None
    instance_factory = None
    instance = None

    def __init__(self, deferred, host, port, proxy_addr, auth, instance_factory):
        self.deferred = deferred
        self.host = host
        self.port = port
        self.proxy_addr = proxy_addr
        self.auth = auth
        self.instance_factory = instance_factory

    def connectionMade(self):
        log.debug("HTTPConnectClient: Proxy connection established: %s:%d",
                  log.safe_addr_str(self.proxy_addr.host), self.proxy_addr.port)

        self.sendCommand("CONNECT", "%s:%d" % (self.host, self.port))
        if self.auth:
            self.sendHeader("Proxy-Authorization", self.auth)
        self.endHeaders()

    def connectionLost(self, reason):
        if self.instance:
            self.instance.connectionLost(reason)
        else:
            # Some HTTP proxies (Eg: polipo) are rude and opt to close the
            # connection instead of sending a status code indicating failure.
            self.onConnectionError(ConnectError("Proxy connection closed during setup"))

    def handleEndHeaders(self):
        log.info("HTTPConnectClient: Connected to %s:%d via %s:%d",
                 log.safe_addr_str(self.host), self.port,
                 log.safe_addr_str(self.proxy_addr.host), self.proxy_addr.port)

        self.setRawMode()
        self.instance = self.instance_factory.buildProtocol(self.proxy_addr)
        self.instance.makeConnection(self.transport)
        self.deferred.callback(self.instance)

        tmp = self.clearLineBuffer()
        if tmp:
            self.instance.dataReceived(tmp)

    def handleStatus(self, version, status, message):
        if status != "200":
            self.onConnectionError(ConnectError("Proxy returned status: %s" % status))

    def rawDataReceived(self, data):
        log.debug("HTTPConnectClient: Received %d bytes of proxied data", len(data))
        if self.instance:
            self.instance.dataReceived(data)
        else:
            raise RuntimeError("HTTPConnectClient.rawDataReceived() called with no instance")

    def onConnectionError(self, reason):
        if self.deferred:
            log.warning("HTTPConnectClient: Connect error: %s", reason)
            self.deferred.errback(reason)
            self.deferred = None
            self.transport.loseConnection()

class HTTPConnectClientFactory(ClientFactory):
    deferred = None
    host = None
    port = None
    auth = None
    instance_factory = None

    def __init__(self, host, port, auth, instance_factory):
        self.deferred = Deferred()
        self.host = host
        self.port = port
        self.auth = auth
        self.instance_factory = instance_factory

    def buildProtocol(self, addr):
        proto = HTTPConnectClient(self.deferred, self.host, self.port, addr, self.auth, self.instance_factory)
        return proto

    def startedConnecting(self, connector):
        self.instance_factory.startedConnectiong(connector)

    def clientConnectionFailed(self, connector, reason):
        self.instance_factory.clientConnectionFailed(connector, reason)

    def clientConnectionLost(self, connector, reason):
        self.instance_factory.clientConnectionLost(connector, reason)

@implementer(IStreamClientEndpoint)
class HTTPConnectClientEndpoint(object):
    host = None
    port = None
    endpoint = None
    auth = None

    def __init__(self, host, port, endpoint, username=None, password=None):
        self.host = host
        self.port = port
        self.endpoint = endpoint
        self.auth = _makeBasicAuth(username, password)

    def connect(self, instance_factory):
        f = HTTPConnectClientFactory(self.host, self.port, self.auth, instance_factory)
        d = self.endpoint.connect(f)
        d.addCallback(lambda proto: f.deferred)
        return d
