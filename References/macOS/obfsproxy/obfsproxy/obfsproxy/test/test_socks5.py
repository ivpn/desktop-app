from twisted.internet import defer, error
from twisted.trial import unittest
from twisted.test import proto_helpers
from twisted.python.failure import Failure

from obfsproxy.network import socks5

import binascii
import struct

class SOCKSv5Protocol_testMethodSelect(unittest.TestCase):

    proto = None
    tr = None

    def _sendMsg(self, msg):
        msg = binascii.unhexlify(msg)
        self.proto.dataReceived(msg)

    def _recvMsg(self, expected):
        self.assertEqual(self.tr.value(), binascii.unhexlify(expected))
        self.tr.clear()

    def setUp(self):
        factory = socks5.SOCKSv5Factory()
        self.proto = factory.buildProtocol(('127.0.0.1', 0))
        self.tr = proto_helpers.StringTransportWithDisconnection()
        self.tr.protocol = self.proto
        self.proto.makeConnection(self.tr)

    def test_InvalidVersion(self):
        """
        Test Method Select message containing a invalid VER.
        """

        # VER = 03, NMETHODS = 01, METHODS = ['00']
        self._sendMsg("030100")
        self.assertFalse(self.tr.connected)

    def test_InvalidNMethods(self):
        """
        Test Method Select message containing no methods.
        """

        # VER = 05, NMETHODS = 00
        self._sendMsg("0500")
        self.assertFalse(self.tr.connected)

    def test_NoAuth(self):
        """
        Test Method Select message containing NO AUTHENTICATION REQUIRED.
        """

        # VER = 05, NMETHODS = 01, METHODS = ['00']
        self._sendMsg("050100")

        # VER = 05, METHOD = 00
        self._recvMsg("0500")
        self.assertEqual(self.proto.authMethod, socks5._SOCKS_AUTH_NO_AUTHENTICATION_REQUIRED)
        self.assertEqual(self.proto.state, self.proto.ST_AUTHENTICATING)
        self.assertTrue(self.tr.connected)

        # Send the first byte of the request to prod it into ST_READ_REQUEST
        self._sendMsg("05")
        self.assertEqual(self.proto.state, self.proto.ST_READ_REQUEST)

    def test_UsernamePasswd(self):
        """
        Test Method Select message containing USERNAME/PASSWORD.
        """

        # VER = 05, NMETHODS = 01, METHODS = ['02']
        self._sendMsg("050102")

        # VER = 05, METHOD = 02
        self._recvMsg("0502")
        self.assertEqual(self.proto.authMethod, socks5._SOCKS_AUTH_USERNAME_PASSWORD)
        self.assertEqual(self.proto.state, self.proto.ST_AUTHENTICATING)
        self.assertTrue(self.tr.connected)

    def test_Both(self):
        """
        Test Method Select message containing both NO AUTHENTICATION REQUIRED
        and USERNAME/PASSWORD.
        """

        # VER = 05, NMETHODS = 02, METHODS = [00, 02]
        self._sendMsg("05020002")

        # VER = 05, METHOD = 02
        self._recvMsg("0502")
        self.assertEqual(self.proto.authMethod, socks5._SOCKS_AUTH_USERNAME_PASSWORD)
        self.assertEqual(self.proto.state, self.proto.ST_AUTHENTICATING)
        self.assertTrue(self.tr.connected)

    def test_Unknown(self):
        """
        Test Method Select message containing a unknown auth method.
        """

        # VER = 05, NMETHODS = 01, METHODS = [01]
        self._sendMsg("050101")

        # VER = 05, METHOD = ff
        self._recvMsg("05ff")
        self.assertEqual(self.proto.authMethod, socks5._SOCKS_AUTH_NO_ACCEPTABLE_METHODS)
        self.assertFalse(self.tr.connected)

    def test_BothUnknown(self):
        """
        Test Method Select message containing supported and unknown methods.
        """

        # VER = 05, NMETHODS = 03, METHODS = [00, 02, ff]
        self._sendMsg("05030002ff")

        # VER = 05, METHOD = 02
        self._recvMsg("0502")
        self.assertEqual(self.proto.authMethod, socks5._SOCKS_AUTH_USERNAME_PASSWORD)
        self.assertEqual(self.proto.state, self.proto.ST_AUTHENTICATING)
        self.assertTrue(self.tr.connected)

    def test_TrailingGarbage(self):
        """
        Test Method Select message with a impatient client.
        """

        # VER = 05, NMETHODS = 01, METHODS = ['00'], Garbage= deadbabe
        self._sendMsg("050100deadbabe")

        self.assertFalse(self.tr.connected)

class SOCKSv5Protocol_testRfc1929Auth(unittest.TestCase):

    proto = None
    tr = None

    def _sendMsg(self, msg):
        msg = binascii.unhexlify(msg)
        self.proto.dataReceived(msg)

    def _recvMsg(self, expected):
        self.assertEqual(self.tr.value(), binascii.unhexlify(expected))
        self.tr.clear()

    def _processAuthTrue(self, uname, passwd):
        self.assertEqual(uname, "ABCDE")
        self.assertEqual(passwd, "abcde")
        return True

    def _processAuthFalse(self, uname, passwd):
        self.assertEqual(uname, "ABCDE")
        self.assertEqual(passwd, "abcde")
        return False

    def setUp(self):
        factory = socks5.SOCKSv5Factory()
        self.proto = factory.buildProtocol(('127.0.0.1', 0))
        self.tr = proto_helpers.StringTransportWithDisconnection()
        self.tr.protocol = self.proto
        self.proto.makeConnection(self.tr)

        # Get things to where the next step is the client sends the auth message
        self._sendMsg("050102")
        self._recvMsg("0502")

    def test_InvalidVersion(self):
        """
        Test auth request containing a invalid VER.
        """

        # VER = 03, ULEN = 5, UNAME = "ABCDE", PLEN = 5, PASSWD = "abcde"
        self._sendMsg("03054142434445056162636465")

        # VER = 01, STATUS = 01
        self._recvMsg("0101")
        self.assertFalse(self.tr.connected)

    def test_InvalidUlen(self):
        """
        Test auth request with a invalid ULEN.
        """

        # VER = 01, ULEN = 0, UNAME = "", PLEN = 5, PASSWD = "abcde"
        self._sendMsg("0100056162636465")

        # VER = 01, STATUS = 01
        self._recvMsg("0101")
        self.assertFalse(self.tr.connected)

    def test_InvalidPlen(self):
        """
        Test auth request with a invalid PLEN.
        """

        # VER = 01, ULEN = 5, UNAME = "ABCDE", PLEN = 0, PASSWD = ""
        self._sendMsg("0105414243444500")

        # VER = 01, STATUS = 01
        self._recvMsg("0101")
        self.assertFalse(self.tr.connected)

    def test_ValidAuthSuccess(self):
        """
        Test auth request that is valid and successful at authenticating.
        """

        self.proto.processRfc1929Auth = self._processAuthTrue

        # VER = 01, ULEN = 5, UNAME = "ABCDE", PLEN = 5, PASSWD = "abcde"
        self._sendMsg("01054142434445056162636465")

        # VER = 01, STATUS = 00
        self._recvMsg("0100")
        self.assertEqual(self.proto.state, self.proto.ST_READ_REQUEST)
        self.assertTrue(self.tr.connected)

    def test_ValidAuthFailure(self):
        """
        Test auth request that is valid and failed at authenticating.
        """

        self.proto.processRfc1929Auth = self._processAuthFalse

        # VER = 01, ULEN = 5, UNAME = "ABCDE", PLEN = 5, PASSWD = "abcde"
        self._sendMsg("01054142434445056162636465")

        # VER = 01, STATUS = 01
        self._recvMsg("0101")
        self.assertFalse(self.tr.connected)

    def test_TrailingGarbage(self):
        """
        Test auth request with a impatient client.
        """

        # VER = 01, ULEN = 5, UNAME = "ABCDE", PLEN = 5, PASSWD = "abcde", Garbage = deadbabe
        self._sendMsg("01054142434445056162636465deadbabe")

        self.assertFalse(self.tr.connected)


class SOCKSv5Protocol_testRequest(unittest.TestCase):

    proto = None
    tr = None
    connectDeferred = None

    def _sendMsg(self, msg):
        msg = binascii.unhexlify(msg)
        self.proto.dataReceived(msg)

    def _recvMsg(self, expected):
        self.assertEqual(self.tr.value(), binascii.unhexlify(expected))
        self.tr.clear()

    def _recvFailureResponse(self, expected):
        # VER = 05, REP = expected, RSV = 00, ATYPE = 01, BND.ADDR = 0.0.0.0, BND.PORT = 0000
        fail_msg = "05" + binascii.hexlify(chr(expected)) + "0001000000000000"
        self._recvMsg(fail_msg)

    def _connectClassIPv4(self, addr, port, klass, *args):
        self.assertEqual(addr, "127.0.0.1")
        self.assertEqual(port, 9050)
        self.connectDeferred = defer.Deferred()
        self.connectDeferred.addCallback(self._connectIPv4)
        return self.connectDeferred

    def _connectIPv4(self, unused):
        self.proto.sendReply(socks5.SOCKSv5Reply.Succeeded, struct.pack("!I", 0x7f000001), 9050)

    def _connectClassIPv6(self, addr, port, klass, *args):
        self.assertEqual(addr, "102:304:506:708:90a:b0c:d0e:f10")
        self.assertEqual(port, 9050)
        self.connectDeferred = defer.Deferred()
        self.connectDeferred.addCallback(self._connectIPv6)
        return self.connectDeferred

    def _connectIPv6(self, unused):
        addr = binascii.unhexlify("0102030405060708090a0b0c0d0e0f10")
        self.proto.sendReply(socks5.SOCKSv5Reply.Succeeded, addr, 9050, socks5._SOCKS_ATYP_IP_V6)

    def _connectClassDomainname(self, addr, port, klass, *args):
        self.assertEqual(addr, "example.com")
        self.assertEqual(port, 9050)
        self.connectDeferred = defer.Deferred()
        self.connectDeferred.addCallback(self._connectIPv4)
        return self.connectDeferred

    def setUp(self):
        factory = socks5.SOCKSv5Factory()
        self.proto = factory.buildProtocol(('127.0.0.1', 0))
        self.tr = proto_helpers.StringTransportWithDisconnection()
        self.tr.protocol = self.proto
        self.proto.makeConnection(self.tr)
        self.connectDeferred = None

        # Get things to where the next step is the client sends the auth message
        self._sendMsg("050100")
        self._recvMsg("0500")

    def test_InvalidVersion(self):
        """
        Test Request with a invalid VER.
        """

        # VER = 03, CMD = 01, RSV = 00, ATYPE = 01, DST.ADDR = 127.0.0.1, DST.PORT = 9050
        self._sendMsg("030100017f000001235a")

        self._recvFailureResponse(socks5.SOCKSv5Reply.GeneralFailure)
        self.assertFalse(self.tr.connected)

    def test_InvalidCommand(self):
        """
        Test Request with a invalid CMD.
        """

        # VER = 05, CMD = 05, RSV = 00, ATYPE = 01, DST.ADDR = 127.0.0.1, DST.PORT = 9050
        self._sendMsg("050500017f000001235a")

        self._recvFailureResponse(socks5.SOCKSv5Reply.CommandNotSupported)
        self.assertFalse(self.tr.connected)

    def test_InvalidRsv(self):
        """
        Test Request with a invalid RSV.
        """

        # VER = 05, CMD = 01, RSV = 30, ATYPE = 01, DST.ADDR = 127.0.0.1, DST.PORT = 9050
        self._sendMsg("050130017f000001235a")

        self._recvFailureResponse(socks5.SOCKSv5Reply.GeneralFailure)
        self.assertFalse(self.tr.connected)

    def test_InvalidAtyp(self):
        """
        Test Request with a invalid ATYP.
        """

        # VER = 05, CMD = 01, RSV = 01, ATYPE = 05, DST.ADDR = 127.0.0.1, DST.PORT = 9050
        self._sendMsg("050100057f000001235a")

        self._recvFailureResponse(socks5.SOCKSv5Reply.AddressTypeNotSupported)
        self.assertFalse(self.tr.connected)

    def test_CmdBind(self):
        """
        Test Request with a BIND CMD.
        """

        # VER = 05, CMD = 02, RSV = 00, ATYPE = 01, DST.ADDR = 127.0.0.1, DST.PORT = 9050
        self._sendMsg("050200017f000001235a")

        self._recvFailureResponse(socks5.SOCKSv5Reply.CommandNotSupported)
        self.assertFalse(self.tr.connected)

    def test_CmdUdpAssociate(self):
        """
        Test Request with a UDP ASSOCIATE CMD.
        """

        # VER = 05, CMD = 03, RSV = 00, ATYPE = 01, DST.ADDR = 127.0.0.1, DST.PORT = 9050
        self._sendMsg("050300017f000001235a")

        self._recvFailureResponse(socks5.SOCKSv5Reply.CommandNotSupported)
        self.assertFalse(self.tr.connected)

    def test_CmdConnectIPv4(self):
        """
        Test Successful Request with a IPv4 CONNECT.
        """

        self.proto.connectClass = self._connectClassIPv4

        # VER = 05, CMD = 01, RSV = 00, ATYPE = 01, DST.ADDR = 127.0.0.1, DST.PORT = 9050
        self._sendMsg("050100017f000001235a")
        self.connectDeferred.callback(self)

        # VER = 05, REP = 00, RSV = 00, ATYPE = 01, BND.ADDR = 127.0.0.1, BND.PORT = 9050
        self._recvMsg("050000017f000001235a")
        self.assertEqual(self.proto.state, self.proto.ST_ESTABLISHED)
        self.assertTrue(self.tr.connected)

    def test_CmdConnectIPv6(self):
        """
        Test Successful Request with a IPv6 CONNECT.
        """

        self.proto.connectClass = self._connectClassIPv6

        # VER = 05, CMD = 01, RSV = 00, ATYPE = 04, DST.ADDR = 0102:0304:0506:0708:090a:0b0c:0d0e:0f10, DST.PORT = 9050
        self._sendMsg("050100040102030405060708090a0b0c0d0e0f10235a")
        self.connectDeferred.callback(self)

        # VER = 05, REP = 00, RSV = 00, ATYPE = 04, BND.ADDR = 0102:0304:0506:0708:090a:0b0c:0d0e:0f10, DST.PORT = 9050
        self._recvMsg("050000040102030405060708090a0b0c0d0e0f10235a")
        self.assertEqual(self.proto.state, self.proto.ST_ESTABLISHED)
        self.assertTrue(self.tr.connected)

    def test_CmdConnectDomainName(self):
        """
        Test Sucessful request with a DOMAINNAME CONNECT.
        """

        self.proto.connectClass = self._connectClassDomainname

        # VER = 05, CMD = 01, RSV = 00, ATYPE = 04, DST.ADDR = example.com, DST.PORT = 9050
        self._sendMsg("050100030b6578616d706c652e636f6d235a")
        self.connectDeferred.callback(self)

        # VER = 05, REP = 00, RSV = 00, ATYPE = 01, BND.ADDR = 127.0.0.1, BND.PORT = 9050
        self._recvMsg("050000017f000001235a")
        self.assertEqual(self.proto.state, self.proto.ST_ESTABLISHED)
        self.assertTrue(self.tr.connected)

    def test_TrailingGarbage(self):
        """
        Test request with a impatient client.
        """

        # VER = 05, CMD = 01, RSV = 00, ATYPE = 01, DST.ADDR = 127.0.0.1, DST.PORT = 9050, Garbage = deadbabe
        self._sendMsg("050100017f000001235adeadbabe")

        self.assertFalse(self.tr.connected)

    def test_CmdConnectErrback(self):
        """
        Test Unsuccessful Request with a IPv4 CONNECT.
        """

        self.proto.connectClass = self._connectClassIPv4

        # VER = 05, CMD = 01, RSV = 00, ATYPE = 01, DST.ADDR = 127.0.0.1, DST.PORT = 9050
        self._sendMsg("050100017f000001235a")
        self.connectDeferred.errback(Failure(error.ConnectionRefusedError("Foo")))

        self._recvFailureResponse(socks5.SOCKSv5Reply.ConnectionRefused)
        self.assertFalse(self.tr.connected)
