#!/usr/bin/python
# -*- coding: utf-8 -*-

"""
The obfs2 module implements the obfs2 protocol.
"""

import random
import hashlib
import argparse
import sys

import obfsproxy.common.aes as aes
import obfsproxy.common.serialize as srlz
import obfsproxy.common.rand as rand

import obfsproxy.transports.base as base

import obfsproxy.common.log as logging

log = logging.get_obfslogger()

MAGIC_VALUE = 0x2BF5CA7E
SEED_LENGTH = 16
MAX_PADDING = 8192
HASH_ITERATIONS = 100000

KEYLEN = 16  # is the length of the key used by E(K,s) -- that is, 16.
IVLEN = 16  # is the length of the IV used by E(K,s) -- that is, 16.

ST_WAIT_FOR_KEY = 0
ST_WAIT_FOR_PADDING = 1
ST_OPEN = 2

def h(x):
    """ H(x) is SHA256 of x. """

    hasher = hashlib.sha256()
    hasher.update(x)
    return hasher.digest()

def hn(x, n):
    """ H^n(x) is H(x) called iteratively n times. """

    data = x
    for _ in xrange(n):
        data = h(data)
    return data

class Obfs2Transport(base.BaseTransport):
    """
    Obfs2Transport implements the obfs2 protocol.
    """

    def __init__(self):
        """Initialize the obfs2 pluggable transport."""
        super(Obfs2Transport, self).__init__()

        # Check if the shared_secret class attribute was already
        # instantiated. If not, instantiate it now.
        if not hasattr(self, 'shared_secret'):
            self.shared_secret = None
        # If external-mode code did not specify the number of hash
        # iterations, just use the default.
        if not hasattr(self, 'ss_hash_iterations'):
            self.ss_hash_iterations = HASH_ITERATIONS

        if self.shared_secret:
            log.debug("Starting obfs2 with shared secret: %s" % self.shared_secret)

        # Our state.
        self.state = ST_WAIT_FOR_KEY

        if self.we_are_initiator:
            self.initiator_seed = rand.random_bytes(SEED_LENGTH) # Initiator's seed.
            self.responder_seed = None # Responder's seed.
        else:
            self.initiator_seed = None # Initiator's seed.
            self.responder_seed = rand.random_bytes(SEED_LENGTH) # Responder's seed

        # Shared secret seed.
        self.secret_seed = None

        # Crypto to encrypt outgoing data.
        self.send_crypto = None
        # Crypto to encrypt outgoing padding.
        self.send_padding_crypto = None
        # Crypto to decrypt incoming data.
        self.recv_crypto = None
        # Crypto to decrypt incoming padding.
        self.recv_padding_crypto = None

        # Number of padding bytes left to read.
        self.padding_left_to_read = 0

        # If it's True, it means that we received upstream data before
        # we had the chance to set up our crypto (after receiving the
        # handshake). This means that when we set up our crypto, we
        # must remember to push the cached upstream data downstream.
        self.pending_data_to_send = False

    @classmethod
    def setup(cls, transport_config):
        """Setup the obfs2 pluggable transport."""
        cls.we_are_initiator = transport_config.weAreClient

        # Check for shared-secret in the server transport options.
        transport_options = transport_config.getServerTransportOptions()
        if transport_options and "shared-secret" in transport_options:
            log.debug("Setting shared-secret from server transport options: '%s'", transport_options["shared-secret"])
            cls.shared_secret = transport_options["shared-secret"]

    @classmethod
    def register_external_mode_cli(cls, subparser):
        subparser.add_argument('--shared-secret', type=str, help='Shared secret')

        # This is a hidden CLI argument for use by the integration
        # tests: so that they don't do an insane amount of hash
        # iterations.
        subparser.add_argument('--ss-hash-iterations', type=int, help=argparse.SUPPRESS)
        super(Obfs2Transport, cls).register_external_mode_cli(subparser)

    @classmethod
    def validate_external_mode_cli(cls, args):
        if args.shared_secret:
            cls.shared_secret = args.shared_secret
        if args.ss_hash_iterations:
            cls.ss_hash_iterations = args.ss_hash_iterations

        try:
            super(Obfs2Transport, cls).validate_external_mode_cli(args)
        except ValueError, err:
            log.error(err)
            sys.exit(1)

    def handle_socks_args(self, args):
        log.debug("obfs2: Got '%s' as SOCKS arguments." % args)

        # A shared secret might already be set if obfsproxy is in
        # external-mode and both a cli shared-secret was specified
        # _and_ a SOCKS per-connection shared secret.
        if self.shared_secret:
            log.notice("obfs2: Hm. Weird configuration. A shared secret "
                       "was specified twice. I will keep the one "
                       "supplied by the SOCKS arguments.")

        if len(args) != 1:
            err_msg = "obfs2: Too many SOCKS arguments (%d) (%s)" % (len(args), str(args))
            log.warning(err_msg)
            raise base.SOCKSArgsError(err_msg)

        if not args[0].startswith("shared-secret="):
            err_msg = "obfs2: SOCKS arg is not correctly formatted  (%s)" % args[0]
            log.warning(err_msg)
            raise base.SOCKSArgsError(err_msg)

        self.shared_secret = args[0][14:]

    def circuitConnected(self):
        """
        Do the obfs2 handshake:
        SEED | E_PAD_KEY( UINT32(MAGIC_VALUE) | UINT32(PADLEN) | WR(PADLEN) )
        """
        # Generate keys for outgoing padding.
        self.send_padding_crypto = \
            self._derive_padding_crypto(self.initiator_seed if self.we_are_initiator else self.responder_seed,
                                        self.send_pad_keytype)

        padding_length = random.randint(0, MAX_PADDING)
        seed = self.initiator_seed if self.we_are_initiator else self.responder_seed

        handshake_message = seed + self.send_padding_crypto.crypt(srlz.htonl(MAGIC_VALUE) +
                                                                  srlz.htonl(padding_length) +
                                                                  rand.random_bytes(padding_length))

        log.debug("obfs2 handshake: %s queued %d bytes (padding_length: %d).",
                  "initiator" if self.we_are_initiator else "responder",
                  len(handshake_message), padding_length)

        self.circuit.downstream.write(handshake_message)

    def receivedUpstream(self, data):
        """
        Got data from upstream. We need to obfuscated and proxy them downstream.
        """
        if not self.send_crypto:
            log.debug("Got upstream data before doing handshake. Caching.")
            self.pending_data_to_send = True
            return

        log.debug("obfs2 receivedUpstream: Transmitting %d bytes.", len(data))
        # Encrypt and proxy them.
        self.circuit.downstream.write(self.send_crypto.crypt(data.read()))

    def receivedDownstream(self, data):
        """
        Got data from downstream. We need to de-obfuscate them and
        proxy them upstream.
        """
        log_prefix = "obfs2 receivedDownstream" # used in logs

        if self.state == ST_WAIT_FOR_KEY:
            log.debug("%s: Waiting for key." % log_prefix)
            if len(data) < SEED_LENGTH + 8:
                log.debug("%s: Not enough bytes for key (%d)." % (log_prefix, len(data)))
                return data # incomplete

            if self.we_are_initiator:
                self.responder_seed = data.read(SEED_LENGTH)
            else:
                self.initiator_seed = data.read(SEED_LENGTH)

            # Now that we got the other seed, let's set up our crypto.
            self.send_crypto = self._derive_crypto(self.send_keytype)
            self.recv_crypto = self._derive_crypto(self.recv_keytype)
            self.recv_padding_crypto = \
                self._derive_padding_crypto(self.responder_seed if self.we_are_initiator else self.initiator_seed,
                                            self.recv_pad_keytype)

            # XXX maybe faster with a single d() instead of two.
            magic = srlz.ntohl(self.recv_padding_crypto.crypt(data.read(4)))
            padding_length = srlz.ntohl(self.recv_padding_crypto.crypt(data.read(4)))

            log.debug("%s: Got %d bytes of handshake data (padding_length: %d, magic: %s)" % \
                          (log_prefix, len(data), padding_length, hex(magic)))

            if magic != MAGIC_VALUE:
                raise base.PluggableTransportError("obfs2: Corrupted magic value '%s'" % hex(magic))
            if padding_length > MAX_PADDING:
                raise base.PluggableTransportError("obfs2: Too big padding length '%s'" % padding_length)

            self.padding_left_to_read = padding_length
            self.state = ST_WAIT_FOR_PADDING

        while self.padding_left_to_read:
            if not data: return

            n_to_drain = self.padding_left_to_read
            if (self.padding_left_to_read > len(data)):
                n_to_drain = len(data)

            data.drain(n_to_drain)
            self.padding_left_to_read -= n_to_drain
            log.debug("%s: Consumed %d bytes of padding, %d still to come (%d).",
                      log_prefix, n_to_drain, self.padding_left_to_read, len(data))

        self.state = ST_OPEN
        log.debug("%s: Processing %d bytes of application data.",
                  log_prefix, len(data))

        if self.pending_data_to_send:
            log.debug("%s: We got pending data to send and our crypto is ready. Pushing!" % log_prefix)
            self.receivedUpstream(self.circuit.upstream.buffer) # XXX touching guts of network.py
            self.pending_data_to_send = False

        self.circuit.upstream.write(self.recv_crypto.crypt(data.read()))

    def _derive_crypto(self, pad_string): # XXX consider secret_seed
        """
        Derive and return an obfs2 key using the pad string in 'pad_string'.
        """
        secret = self.mac(pad_string,
                          self.initiator_seed + self.responder_seed,
                          self.shared_secret)
        return aes.AES_CTR_128(secret[:KEYLEN], secret[KEYLEN:],
                               counter_wraparound=True)

    def _derive_padding_crypto(self, seed, pad_string): # XXX consider secret_seed
        """
        Derive and return an obfs2 padding key using the pad string in 'pad_string'.
        """
        secret = self.mac(pad_string,
                          seed,
                          self.shared_secret)
        return aes.AES_CTR_128(secret[:KEYLEN], secret[KEYLEN:],
                               counter_wraparound=True)

    def mac(self, s, x, secret):
        """
        obfs2 regular MAC: MAC(s, x) = H(s | x | s)

        Optionally, if the client and server share a secret value SECRET,
        they can replace the MAC function with:
        MAC(s,x) = H^n(s | x | H(SECRET) | s)

        where n = HASH_ITERATIONS.
        """
        if secret:
            secret_hash = h(secret)
            return hn(s + x + secret_hash + s, self.ss_hash_iterations)
        else:
            return h(s + x + s)


class Obfs2Client(Obfs2Transport):

    """
    Obfs2Client is a client for the obfs2 protocol.
    The client and server differ in terms of their padding strings.
    """

    def __init__(self):
        self.send_pad_keytype = 'Initiator obfuscation padding'
        self.recv_pad_keytype = 'Responder obfuscation padding'
        self.send_keytype = "Initiator obfuscated data"
        self.recv_keytype = "Responder obfuscated data"

        Obfs2Transport.__init__(self)


class Obfs2Server(Obfs2Transport):

    """
    Obfs2Server is a server for the obfs2 protocol.
    The client and server differ in terms of their padding strings.
    """

    def __init__(self):
        self.send_pad_keytype = 'Responder obfuscation padding'
        self.recv_pad_keytype = 'Initiator obfuscation padding'
        self.send_keytype = "Responder obfuscated data"
        self.recv_keytype = "Initiator obfuscated data"

        Obfs2Transport.__init__(self)
