import os

from twisted.internet import reactor

import obfsproxy.common.log as logging
import obfsproxy.common.serialize as srlz
import obfsproxy.common.hmac_sha256 as hmac_sha256
import obfsproxy.common.rand as rand

import obfsproxy.network.network as network

log = logging.get_obfslogger()

# Authentication states:
STATE_WAIT_FOR_AUTH_TYPES = 1
STATE_WAIT_FOR_SERVER_NONCE = 2
STATE_WAIT_FOR_AUTH_RESULTS = 3
STATE_WAIT_FOR_OKAY = 4
STATE_OPEN = 5

# Authentication protocol parameters:
AUTH_PROTOCOL_HEADER_LEN = 4

# Safe-cookie authentication parameters:
AUTH_SERVER_TO_CLIENT_CONST = "ExtORPort authentication server-to-client hash"
AUTH_CLIENT_TO_SERVER_CONST = "ExtORPort authentication client-to-server hash"
AUTH_NONCE_LEN = 32
AUTH_HASH_LEN = 32

# Extended ORPort commands:
# Transport-to-Bridge
EXT_OR_CMD_TB_DONE = 0x0000
EXT_OR_CMD_TB_USERADDR = 0x0001
EXT_OR_CMD_TB_TRANSPORT = 0x0002

# Bridge-to-Transport
EXT_OR_CMD_BT_OKAY = 0x1000
EXT_OR_CMD_BT_DENY = 0x1001
EXT_OR_CMD_BT_CONTROL = 0x1002

# Authentication cookie parameters
AUTH_COOKIE_LEN = 32
AUTH_COOKIE_HEADER_LEN = 32
AUTH_COOKIE_FILE_LEN = AUTH_COOKIE_LEN + AUTH_COOKIE_HEADER_LEN
AUTH_COOKIE_HEADER = "! Extended ORPort Auth Cookie !\x0a"

def _read_auth_cookie(cookie_path):
    """
    Read an Extended ORPort authentication cookie from 'cookie_path' and return it.
    Throw CouldNotReadCookie if we couldn't read the cookie.
    """

    # Check if file exists.
    if not os.path.exists(cookie_path):
        raise CouldNotReadCookie("'%s' doesn't exist" % cookie_path)

    # Check its size and make sure it's correct before opening.
    auth_cookie_file_size = os.path.getsize(cookie_path)
    if auth_cookie_file_size != AUTH_COOKIE_FILE_LEN:
        raise CouldNotReadCookie("Cookie '%s' is the wrong size (%i bytes instead of %d)" % \
                                     (cookie_path, auth_cookie_file_size, AUTH_COOKIE_FILE_LEN))

    try:
        with file(cookie_path, 'rb', 0) as f:
            header = f.read(AUTH_COOKIE_HEADER_LEN) # first 32 bytes are the header

            if header != AUTH_COOKIE_HEADER:
                raise CouldNotReadCookie("Corrupted cookie file header '%s'." % header)

            return f.read(AUTH_COOKIE_LEN) # nexta 32 bytes should be the cookie.

    except IOError, exc:
        raise CouldNotReadCookie("Unable to read '%s' (%s)" % (cookie_path, exc))

class ExtORPortProtocol(network.GenericProtocol):
    """
    Represents a connection to the Extended ORPort. It begins by
    completing the Extended ORPort authentication, then sending some
    Extended ORPort commands, and finally passing application-data
    like it would do to an ORPort.

    Specifically, after completing the Extended ORPort authentication
    we send a USERADDR command with the address of our client, a
    TRANSPORT command with the name of the pluggable transport, and a
    DONE command to signal that we are done with the Extended ORPort
    protocol. Then we wait for an OKAY command back from the server to
    start sending application-data.

    Attributes:
    state: The protocol state the connections is currently at.
    ext_orport_addr: The address of the Extended ORPort.

    peer_addr: The address of the client, in the other side of the
               circuit, that connected to our downstream side.
    cookie_file: Path to the Extended ORPort authentication cookie.
    client_nonce: A random nonce used in the Extended ORPort
                  authentication protocol.
    client_hash: Our hash which is used to verify our knowledge of the
                 authentication cookie in the Extended ORPort Authentication
                 protocol.
    """
    def __init__(self, circuit, ext_orport_addr, cookie_file, peer_addr, transport_name):
        self.state = STATE_WAIT_FOR_AUTH_TYPES
        self.name = "ext_%s" % hex(id(self))

        self.ext_orport_addr = ext_orport_addr
        self.peer_addr = peer_addr
        self.cookie_file = cookie_file

        self.client_nonce = rand.random_bytes(AUTH_NONCE_LEN)
        self.client_hash = None

        self.transport_name = transport_name

        network.GenericProtocol.__init__(self, circuit)

    def connectionMade(self):
        pass

    def dataReceived(self, data_rcvd):
        """
        We got some data, process it according to our current state.
        """
        if self.closed:
            log.debug("%s: ExtORPort dataReceived called while closed. Ignoring.", self.name)
            return

        self.buffer.write(data_rcvd)

        if self.state == STATE_WAIT_FOR_AUTH_TYPES:
            try:
                self._handle_auth_types()
            except NeedMoreData:
                return
            except UnsupportedAuthTypes, err:
                log.warning("Extended ORPort Cookie Authentication failed: %s" % err)
                self.close()
                return

            self.state = STATE_WAIT_FOR_SERVER_NONCE

        if self.state == STATE_WAIT_FOR_SERVER_NONCE:
            try:
                self._handle_server_nonce_and_hash()
            except NeedMoreData:
                return
            except (CouldNotReadCookie, RcvdInvalidAuth) as err:
                log.warning("Extended ORPort Cookie Authentication failed: %s" % err)
                self.close()
                return

            self.state = STATE_WAIT_FOR_AUTH_RESULTS

        if self.state == STATE_WAIT_FOR_AUTH_RESULTS:
            try:
                self._handle_auth_results()
            except NeedMoreData:
                return
            except AuthFailed, err:
                log.warning("Extended ORPort Cookie Authentication failed: %s" % err)
                self.close()
                return

            # We've finished the Extended ORPort authentication
            # protocol. Now send all the Extended ORPort commands we
            # want to send.
            try:
                self._send_ext_orport_commands()
            except CouldNotWriteExtCommand:
                self.close()
                return

            self.state = STATE_WAIT_FOR_OKAY

        if self.state == STATE_WAIT_FOR_OKAY:
            try:
                self._handle_okay()
            except NeedMoreData:
                return
            except ExtORPortProtocolFailed as err:
                log.warning("Extended ORPort Cookie Authentication failed: %s" % err)
                self.close()
                return

            self.state = STATE_OPEN

        if self.state == STATE_OPEN:
            # We are done with the Extended ORPort protocol, we now
            # treat the Extended ORPort as a normal ORPort.
            if not self.circuit.circuitIsReady():
                self.circuit.setUpstreamConnection(self)
            self.circuit.dataReceived(self.buffer, self)

    def _send_ext_orport_commands(self):
        """
        Send all the Extended ORPort commands we want to send.

        Throws CouldNotWriteExtCommand.
        """

        # Send the actual IP address of our client to the Extended
        # ORPort, then signal that we are done and that we want to
        # start transferring application-data.
        self._write_ext_orport_command(EXT_OR_CMD_TB_USERADDR, '%s:%s' % (self.peer_addr.host, self.peer_addr.port))
        self._write_ext_orport_command(EXT_OR_CMD_TB_TRANSPORT, '%s' % self.transport_name)
        self._write_ext_orport_command(EXT_OR_CMD_TB_DONE, '')

    def _handle_auth_types(self):
        """
        Read authentication types that the server supports, select
        one, and send it to the server.

        Throws NeedMoreData and UnsupportedAuthTypes.
        """

        if len(self.buffer) < 2:
            raise NeedMoreData('Not enough data')

        data = self.buffer.peek()
        if '\x00' not in data: # haven't received EndAuthTypes yet
            log.debug("%s: Got some auth types data but no EndAuthTypes yet." % self.name)
            raise NeedMoreData('Not EndAuthTypes.')

        # Drain all data up to (and including) the EndAuthTypes.
        log.debug("%s: About to drain %d bytes from %d." % \
                        (self.name, data.index('\x00')+1, len(self.buffer)))
        data = self.buffer.read(data.index('\x00')+1)

        if '\x01' not in data:
            raise UnsupportedAuthTypes("%s: Could not find supported auth type (%s)." % (self.name, repr(data)))

        # Send back chosen auth type.
        self.write("\x01") # Static, since we only support auth type '1' atm.

        # Since we are doing the safe-cookie protocol, now send our
        # nonce.
        # XXX This will need to be refactored out of this function in
        # the future, when we have more than one auth types.
        self.write(self.client_nonce)

    def _handle_server_nonce_and_hash(self):
        """
        Get the server's nonce and hash, validate them and send our own hash.

        Throws NeedMoreData and RcvdInvalidAuth and CouldNotReadCookie.
        """

        if len(self.buffer) < AUTH_HASH_LEN + AUTH_NONCE_LEN:
            raise NeedMoreData('Need more data')

        server_hash = self.buffer.read(AUTH_HASH_LEN)
        server_nonce = self.buffer.read(AUTH_NONCE_LEN)
        auth_cookie = _read_auth_cookie(self.cookie_file)

        proper_server_hash = hmac_sha256.hmac_sha256_digest(auth_cookie,
                                                            AUTH_SERVER_TO_CLIENT_CONST + self.client_nonce + server_nonce)

        log.debug("%s: client_nonce: %s\nserver_nonce: %s\nserver_hash: %s\nproper_server_hash: %s\n" % \
                      (self.name, repr(self.client_nonce), repr(server_nonce), repr(server_hash), repr(proper_server_hash)))

        if proper_server_hash != server_hash:
            raise RcvdInvalidAuth("%s: Invalid server hash. Authentication failed." % (self.name))

        client_hash = hmac_sha256.hmac_sha256_digest(auth_cookie,
                                                     AUTH_CLIENT_TO_SERVER_CONST + self.client_nonce + server_nonce)

        # Send our hash.
        self.write(client_hash)

    def _handle_auth_results(self):
        """
        Get the authentication results. See if the authentication
        succeeded or failed, and take appropriate actions.

        Throws NeedMoreData and AuthFailed.
        """
        if len(self.buffer) < 1:
            raise NeedMoreData("Not enough data for body.")

        result = self.buffer.read(1)
        if result != '\x01':
            raise AuthFailed("%s: Authentication failed (%s)!" % (self.name, repr(result)))

        log.debug("%s: Authentication successful!" % self.name)

    def _handle_okay(self):
        """
        We've sent a DONE command to the Extended ORPort and we
        now check if the Extended ORPort liked it or not.

        Throws NeedMoreData and ExtORPortProtocolFailed.
        """

        cmd, _ = self._get_ext_orport_command(self.buffer)
        if cmd != EXT_OR_CMD_BT_OKAY:
            raise ExtORPortProtocolFailed("%s: Unexpected command received (%d) after sending DONE." % (self.name, cmd))

    def _get_ext_orport_command(self, buf):
        """
        Reads an Extended ORPort command from 'buf'. Returns (command,
        body) if it was well-formed, where 'command' is the Extended
        ORPort command type, and 'body' is its body.

        Throws NeedMoreData.
        """
        if len(buf) < AUTH_PROTOCOL_HEADER_LEN:
            raise NeedMoreData("Not enough data for header.")

        header = buf.peek(AUTH_PROTOCOL_HEADER_LEN)
        cmd = srlz.ntohs(header[:2])
        bodylen = srlz.ntohs(header[2:4])

        if (bodylen > len(buf) - AUTH_PROTOCOL_HEADER_LEN): # Not all here yet
            raise NeedMoreData("Not enough data for body.")

        # We have a whole command. Drain the header.
        buf.drain(4)
        body = buf.read(bodylen)

        return (cmd, body)

    def _write_ext_orport_command(self, command, body):
        """
        Serialize 'command' and 'body' to an Extended ORPort command
        and send it to the Extended ORPort.

        Throws CouldNotWriteExtCommand
        """
        payload = ''

        if len(body) > 65535: # XXX split instead of quitting?
            log.warning("Obfsproxy was asked to send Extended ORPort command with more than "
                        "65535 bytes of body. This is not supported by the Extended ORPort "
                        "protocol. Please file a bug.")
            raise CouldNotWriteExtCommand("Too large body.")
        if command > 65535:
            raise CouldNotWriteExtCommand("Not supported command type.")

        payload += srlz.htons(command)
        payload += srlz.htons(len(body))
        payload += body # body might be absent (empty string)
        self.write(payload)


class ExtORPortClientFactory(network.StaticDestinationClientFactory):
    def __init__(self, circuit, cookie_file, peer_addr, transport_name):
        self.circuit = circuit
        self.peer_addr = peer_addr
        self.cookie_file = cookie_file
        self.transport_name = transport_name

        self.name = "fact_ext_c_%s" % hex(id(self))

    def buildProtocol(self, addr):
        return ExtORPortProtocol(self.circuit, addr, self.cookie_file, self.peer_addr, self.transport_name)

class ExtORPortServerFactory(network.StaticDestinationClientFactory):
    def __init__(self, ext_or_addrport, ext_or_cookie_file, transport_name, transport_class, pt_config):
        self.ext_or_host = ext_or_addrport[0]
        self.ext_or_port = ext_or_addrport[1]
        self.cookie_file = ext_or_cookie_file

        self.transport_name = transport_name
        self.transport_class = transport_class
        self.pt_config = pt_config

        self.name = "fact_ext_s_%s" % hex(id(self))

    def startFactory(self):
        log.debug("%s: Starting up Extended ORPort server factory." % self.name)

    def buildProtocol(self, addr):
        log.debug("%s: New connection from %s:%d." % (self.name, log.safe_addr_str(addr.host), addr.port))

        circuit = network.Circuit(self.transport_class())

        # XXX instantiates a new factory for each client
        clientFactory = ExtORPortClientFactory(circuit, self.cookie_file, addr, self.transport_name)
        reactor.connectTCP(self.ext_or_host, self.ext_or_port, clientFactory)

        return network.StaticDestinationProtocol(circuit, 'server', addr)

# XXX Exceptions need more thought and work. Most of these can be generalized.
class RcvdInvalidAuth(Exception): pass
class AuthFailed(Exception): pass
class UnsupportedAuthTypes(Exception): pass
class ExtORPortProtocolFailed(Exception): pass
class CouldNotWriteExtCommand(Exception): pass
class CouldNotReadCookie(Exception): pass
class NeedMoreData(Exception): pass
