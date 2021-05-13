import unittest

import os
import base64
import shutil
import tempfile

import Crypto.Hash.SHA256
import Crypto.Hash.HMAC

import obfsproxy.common.log as logging
import obfsproxy.network.buffer as obfs_buf
import obfsproxy.common.transport_config as transport_config
import obfsproxy.transports.base as base

import obfsproxy.transports.scramblesuit.state as state
import obfsproxy.transports.scramblesuit.util as util
import obfsproxy.transports.scramblesuit.const as const
import obfsproxy.transports.scramblesuit.mycrypto as mycrypto
import obfsproxy.transports.scramblesuit.uniformdh as uniformdh
import obfsproxy.transports.scramblesuit.scramblesuit as scramblesuit
import obfsproxy.transports.scramblesuit.message as message
import obfsproxy.transports.scramblesuit.state as state
import obfsproxy.transports.scramblesuit.ticket as ticket
import obfsproxy.transports.scramblesuit.packetmorpher as packetmorpher
import obfsproxy.transports.scramblesuit.probdist as probdist


# Disable all logging as it would yield plenty of warning and error
# messages.
log = logging.get_obfslogger()
log.disable_logs()

class CryptoTest( unittest.TestCase ):

    """
    The HKDF test cases are taken from the appendix of RFC 5869:
    https://tools.ietf.org/html/rfc5869
    """

    def setUp( self ):
        pass

    def extract( self, salt, ikm ):
        return Crypto.Hash.HMAC.new(salt, ikm, Crypto.Hash.SHA256).digest()

    def runHKDF( self, ikm, salt, info, prk, okm ):
        myprk = self.extract(salt, ikm)
        self.failIf(myprk != prk)
        myokm = mycrypto.HKDF_SHA256(myprk, info).expand()
        self.failUnless(myokm in okm)

    def test1_HKDF_TestCase1( self ):

        ikm = "0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b".decode('hex')
        salt = "000102030405060708090a0b0c".decode('hex')
        info = "f0f1f2f3f4f5f6f7f8f9".decode('hex')
        prk = ("077709362c2e32df0ddc3f0dc47bba6390b6c73bb50f9c3122e" + \
              "c844ad7c2b3e5").decode('hex')
        okm = ("3cb25f25faacd57a90434f64d0362f2a2d2d0a90cf1a5a4c5db" + \
              "02d56ecc4c5bf34007208d5b887185865").decode('hex')

        self.runHKDF(ikm, salt, info, prk, okm)

    def test2_HKDF_TestCase2( self ):

        ikm = ("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c" + \
               "1d1e1f202122232425262728292a2b2c2d2e2f30313233343536373839" + \
               "3a3b3c3d3e3f404142434445464748494a4b4c4d4e4f").decode('hex')
        salt =("606162636465666768696a6b6c6d6e6f707172737475767778797a7b7c" + \
               "7d7e7f808182838485868788898a8b8c8d8e8f90919293949596979899" + \
               "9a9b9c9d9e9fa0a1a2a3a4a5a6a7a8a9aaabacadaeaf").decode('hex')
        info =("b0b1b2b3b4b5b6b7b8b9babbbcbdbebfc0c1c2c3c4c5c6c7c8c9cacbcc" + \
               "cdcecfd0d1d2d3d4d5d6d7d8d9dadbdcdddedfe0e1e2e3e4e5e6e7e8e9" + \
               "eaebecedeeeff0f1f2f3f4f5f6f7f8f9fafbfcfdfeff").decode('hex')
        prk = ("06a6b88c5853361a06104c9ceb35b45cef760014904671014a193f40c1" + \
               "5fc244").decode('hex')
        okm = ("b11e398dc80327a1c8e7f78c596a49344f012eda2d4efad8a050cc4c19" + \
               "afa97c59045a99cac7827271cb41c65e590e09da3275600c2f09b83677" + \
               "93a9aca3db71cc30c58179ec3e87c14c01d5c1" + \
               "f3434f1d87").decode('hex')

        self.runHKDF(ikm, salt, info, prk, okm)

    def test3_HKDF_TestCase3( self ):
        ikm = "0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b".decode('hex')
        salt = ""
        info = ""
        prk = ("19ef24a32c717b167f33a91d6f648bdf96596776afdb6377a" + \
               "c434c1c293ccb04").decode('hex')
        okm = ("8da4e775a563c18f715f802a063c5a31b8a11f5c5ee1879ec" + \
               "3454e5f3c738d2d9d201395faa4b61a96c8").decode('hex')

        self.runHKDF(ikm, salt, info, prk, okm)

    def test4_HKDF_TestCase4( self ):

        self.assertRaises(ValueError,
                          mycrypto.HKDF_SHA256, "x" * 40, length=(32*255)+1)

        self.assertRaises(ValueError,
                          mycrypto.HKDF_SHA256, "tooShort")

        # Accidental re-use should raise an exception.
        hkdf = mycrypto.HKDF_SHA256("x" * 40)
        hkdf.expand()
        self.assertRaises(base.PluggableTransportError, hkdf.expand)

    def test4_CSPRNG( self ):
        self.failIf(mycrypto.strongRandom(10) == mycrypto.strongRandom(10))
        self.failIf(len(mycrypto.strongRandom(100)) != 100)

    def test5_AES( self ):
        plain = "this is a test"
        key = os.urandom(16)
        iv = os.urandom(8)

        crypter1 = mycrypto.PayloadCrypter()
        crypter1.setSessionKey(key, iv)
        crypter2 = mycrypto.PayloadCrypter()
        crypter2.setSessionKey(key, iv)

        cipher = crypter1.encrypt(plain)

        self.failIf(cipher == plain)
        self.failUnless(crypter2.decrypt(cipher) == plain)

    def test6_HMAC_SHA256_128( self ):
        self.assertRaises(AssertionError, mycrypto.HMAC_SHA256_128,
                          "x" * (const.SHARED_SECRET_LENGTH - 1), "test")

        self.failUnless(len(mycrypto.HMAC_SHA256_128("x" * \
                        const.SHARED_SECRET_LENGTH, "test")) == 16)


class UniformDHTest( unittest.TestCase ):

    def setUp( self ):
        weAreServer = True
        self.udh = uniformdh.new("A" * const.SHARED_SECRET_LENGTH, weAreServer)

    def test1_createHandshake( self ):
        handshake = self.udh.createHandshake()
        self.failUnless((const.PUBLIC_KEY_LENGTH +
                         const.MARK_LENGTH +
                         const.HMAC_SHA256_128_LENGTH) <= len(handshake) <=
                        (const.MARK_LENGTH +
                         const.HMAC_SHA256_128_LENGTH +
                         const.MAX_PADDING_LENGTH))

    def test2_receivePublicKey( self ):
        buf = obfs_buf.Buffer(self.udh.createHandshake())

        def callback( masterKey ):
            self.failUnless(len(masterKey) == const.MASTER_KEY_LENGTH)

        self.failUnless(self.udh.receivePublicKey(buf, callback) == True)

        publicKey = self.udh.getRemotePublicKey()
        self.failUnless(len(publicKey) == const.PUBLIC_KEY_LENGTH)

    def test3_invalidHMAC( self ):
        # Make the HMAC invalid.
        handshake = self.udh.createHandshake()
        if handshake[-1] != 'a':
            handshake = handshake[:-1] + 'a'
        else:
            handshake = handshake[:-1] + 'b'

        buf = obfs_buf.Buffer(handshake)

        self.failIf(self.udh.receivePublicKey(buf, lambda x: x) == True)

    def test4_extractPublicKey( self ):

        # Create UniformDH authentication message.
        sharedSecret = "A" * const.SHARED_SECRET_LENGTH

        realEpoch = util.getEpoch

        # Try three valid and one invalid epoch value.
        for epoch in util.expandedEpoch() + ["000000"]:
            udh = uniformdh.new(sharedSecret, True)

            util.getEpoch = lambda: epoch
            authMsg = udh.createHandshake()
            util.getEpoch = realEpoch

            buf = obfs_buf.Buffer()
            buf.write(authMsg)

            if epoch == "000000":
                self.assertFalse(udh.extractPublicKey(buf))
            else:
                self.assertTrue(udh.extractPublicKey(buf))


class UtilTest( unittest.TestCase ):

    def test1_isValidHMAC( self ):
        self.failIf(util.isValidHMAC("A" * const.HMAC_SHA256_128_LENGTH,
                                     "B" * const.HMAC_SHA256_128_LENGTH,
                                     "X" * const.SHA256_LENGTH) == True)
        self.failIf(util.isValidHMAC("A" * const.HMAC_SHA256_128_LENGTH,
                                     "A" * const.HMAC_SHA256_128_LENGTH,
                                     "X" * const.SHA256_LENGTH) == False)

    def test2_locateMark( self ):
        self.failIf(util.locateMark("D", "ABC") != None)

        hmac = "X" * const.HMAC_SHA256_128_LENGTH
        mark = "A" * const.MARK_LENGTH
        payload = mark + hmac

        self.failIf(util.locateMark(mark, payload) == None)
        self.failIf(util.locateMark(mark, payload[:-1]) != None)

    def test3_sanitiseBase32( self ):
        self.failUnless(util.sanitiseBase32("abc") == "ABC")
        self.failUnless(util.sanitiseBase32("ABC1XYZ") == "ABCIXYZ")
        self.failUnless(util.sanitiseBase32("ABC1XYZ0") == "ABCIXYZO")

    def test4_setStateLocation( self ):
        name = (const.TRANSPORT_NAME).lower()

        # Check if function creates non-existant directories.
        d = tempfile.mkdtemp()
        util.setStateLocation(d)
        self.failUnless(const.STATE_LOCATION == "%s/%s/" % (d, name))
        self.failUnless(os.path.exists("%s/%s/" % (d, name)))

        # Nothing should change if we pass "None".
        util.setStateLocation(None)
        self.failUnless(const.STATE_LOCATION == "%s/%s/" % (d, name))

        shutil.rmtree(d)

    def test5_getEpoch( self ):
        e = util.getEpoch()
        self.failUnless(isinstance(e, basestring))

    def test7_readFromFile( self ):

        # Read from non-existant file.
        self.failUnless(util.readFromFile(tempfile.mktemp()) == None)

        # Read file where we (hopefully) don't have permissions.
        self.failUnless(util.readFromFile("/etc/shadow") == None)

class StateTest( unittest.TestCase ):
    def setUp( self ):
        const.STATE_LOCATION = tempfile.mkdtemp()
        self.stateFile = os.path.join(const.STATE_LOCATION, const.SERVER_STATE_FILE)
        self.state = state.State()

    def tearDown( self ):
        try:
            shutil.rmtree(const.STATE_LOCATION)
        except OSError:
            pass

    def test1_genState( self ):
        self.state.genState()
        self.failUnless(os.path.exists(self.stateFile))

    def test2_loadState( self ):
        # load() should create the state file if it doesn't exist yet.
        self.failIf(os.path.exists(self.stateFile))
        self.failUnless(isinstance(state.load(), state.State))
        self.failUnless(os.path.exists(self.stateFile))

    def test3_replay( self ):
        key = "A" * const.HMAC_SHA256_128_LENGTH
        self.state.genState()
        self.state.registerKey(key)
        self.failUnless(self.state.isReplayed(key))
        self.failIf(self.state.isReplayed("B" * const.HMAC_SHA256_128_LENGTH))

    def test4_ioerrorFail( self ):
        def fake_open(name, mode):
            raise IOError()
        self.state.genState()

        import __builtin__
        real_open = __builtin__.open
        __builtin__.open = fake_open

        # Make state.load() fail
        self.assertRaises(SystemExit, state.load)
        # Make State.writeState() fail.
        self.assertRaises(SystemExit, self.state.genState)

        __builtin__.open = real_open

class MockArgs( object ):
    uniformDHSecret = sharedSecret = ext_cookie_file = dest = None
    mode = 'socks'


class ScrambleSuitTransportTest( unittest.TestCase ):

    def setUp( self ):
        config = transport_config.TransportConfig( )
        config.state_location = const.STATE_LOCATION
        args = MockArgs( )
        suit = scramblesuit.ScrambleSuitTransport
        suit.weAreServer = False

        self.suit = suit
        self.args = args
        self.config = config

        self.validSecret = base64.b32encode( 'A' * const.SHARED_SECRET_LENGTH )
        self.invalidSecret = 'a' * const.SHARED_SECRET_LENGTH

        self.statefile = tempfile.mkdtemp()

    def tearDown( self ):
        try:
            shutil.rmtree(self.statefile)
        except OSError:
            pass

    def test1_validateExternalModeCli( self ):
        """Test with valid scramblesuit args and valid obfsproxy args."""
        self.args.uniformDHSecret = self.validSecret

        self.assertTrue(
            super( scramblesuit.ScrambleSuitTransport,
                   self.suit ).validate_external_mode_cli( self.args ))

        self.assertIsNone( self.suit.validate_external_mode_cli( self.args ) )

    def test2_validateExternalModeCli( self ):
        """Test with invalid scramblesuit args and valid obfsproxy args."""
        self.args.uniformDHSecret = self.invalidSecret

        with self.assertRaises( base.PluggableTransportError ):
            self.suit.validate_external_mode_cli( self.args )

    def test3_get_public_server_options( self ):
        transCfg = transport_config.TransportConfig()
        transCfg.setStateLocation(self.statefile)

        scramblesuit.ScrambleSuitTransport.setup(transCfg)
        options = scramblesuit.ScrambleSuitTransport.get_public_server_options("")
        self.failUnless("password" in options)

        d = { "password": "3X5BIA2MIHLZ55UV4VAEGKZIQPPZ4QT3" }
        options = scramblesuit.ScrambleSuitTransport.get_public_server_options(d)
        self.failUnless("password" in options)
        self.failUnless(options["password"] == "3X5BIA2MIHLZ55UV4VAEGKZIQPPZ4QT3")

class MessageTest( unittest.TestCase ):

    def test1_createProtocolMessages( self ):
        # An empty message consists only of a header.
        self.failUnless(len(message.createProtocolMessages("")[0]) == \
                        const.HDR_LENGTH)

        msg = message.createProtocolMessages('X' * const.MPU)
        self.failUnless((len(msg) == 1) and (len(msg[0]) == const.MTU))

        msg = message.createProtocolMessages('X' * (const.MPU + 1))
        self.failUnless((len(msg) == 2) and \
                        (len(msg[0]) == const.MTU) and \
                        (len(msg[1]) == (const.HDR_LENGTH + 1)))

    def test2_getFlagNames( self ):
        self.failUnless(message.getFlagNames(0) == "Undefined")
        self.failUnless(message.getFlagNames(1) == "PAYLOAD")
        self.failUnless(message.getFlagNames(2) == "NEW_TICKET")
        self.failUnless(message.getFlagNames(4) == "PRNG_SEED")

    def test3_isSane( self ):
        self.failUnless(message.isSane(0, 0, const.FLAG_NEW_TICKET) == True)
        self.failUnless(message.isSane(const.MPU, const.MPU,
                                       const.FLAG_PRNG_SEED) == True)
        self.failUnless(message.isSane(const.MPU + 1, 0,
                                       const.FLAG_PAYLOAD) == False)
        self.failUnless(message.isSane(0, 0, 1234) == False)
        self.failUnless(message.isSane(0, 1, const.FLAG_PAYLOAD) == False)

    def test4_ProtocolMessage( self ):
        flags = [const.FLAG_NEW_TICKET,
                 const.FLAG_PAYLOAD,
                 const.FLAG_PRNG_SEED]

        self.assertRaises(base.PluggableTransportError,
                          message.ProtocolMessage, "1", paddingLen=const.MPU)

class TicketTest( unittest.TestCase ):
    def setUp( self ):
        const.STATE_LOCATION = tempfile.mkdtemp()
        self.stateFile = os.path.join(const.STATE_LOCATION, const.SERVER_STATE_FILE)
        self.state = state.State()
        self.state.genState()

    def tearDown( self ):
        try:
            shutil.rmtree(const.STATE_LOCATION)
        except OSError:
            pass

    def test1_authentication( self ):
        ss = scramblesuit.ScrambleSuitTransport()
        ss.srvState = self.state

        realEpoch = util.getEpoch

        # Try three valid and one invalid epoch value.
        for epoch in util.expandedEpoch() + ["000000"]:

            util.getEpoch = lambda: epoch

            # Prepare ticket message.
            blurb = ticket.issueTicketAndKey(self.state)
            rawTicket = blurb[const.MASTER_KEY_LENGTH:]
            masterKey = blurb[:const.MASTER_KEY_LENGTH]
            ss.deriveSecrets(masterKey)
            ticketMsg = ticket.createTicketMessage(rawTicket, ss.recvHMAC)

            util.getEpoch = realEpoch

            buf = obfs_buf.Buffer()
            buf.write(ticketMsg)

            if epoch == "000000":
                self.assertFalse(ss.receiveTicket(buf))
            else:
                self.assertTrue(ss.receiveTicket(buf))

class PacketMorpher( unittest.TestCase ):

    def test1_calcPadding( self ):

        def checkDistribution( dist ):
            pm = packetmorpher.new(dist)
            for i in xrange(0, const.MTU + 2):
                padLen = pm.calcPadding(i)
                self.assertTrue(const.HDR_LENGTH <= \
                                padLen < \
                                (const.MTU + const.HDR_LENGTH))

        # Test randomly generated distributions.
        for i in xrange(0, 100):
            checkDistribution(None)

        # Test border-case distributions.
        checkDistribution(probdist.new(lambda: 0))
        checkDistribution(probdist.new(lambda: 1))
        checkDistribution(probdist.new(lambda: const.MTU))
        checkDistribution(probdist.new(lambda: const.MTU + 1))

    def test2_getPadding( self ):
        pm = packetmorpher.new()
        sendCrypter = mycrypto.PayloadCrypter()
        sendCrypter.setSessionKey("A" * 32,  "A" * 8)
        sendHMAC = "A" * 32

        for i in xrange(0, const.MTU + 2):
            padLen = len(pm.getPadding(sendCrypter, sendHMAC, i))
            self.assertTrue(const.HDR_LENGTH <= padLen < const.MTU + \
                            const.HDR_LENGTH)


if __name__ == '__main__':
    unittest.main()
