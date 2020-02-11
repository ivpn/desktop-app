import unittest
import time

import obfsproxy.transports.obfs3_dh as obfs3_dh
import twisted.trial.unittest
from twisted.python import log

class testUniformDH_KAT(twisted.trial.unittest.TestCase):
    #
    # Test keypair x/X:
    #
    # The test vector specifies "... 756e" for x but this forces the UniformDH
    # code to return p - X as the public key, and more importantly that's what
    # the original material I used ends with.
    #
    _x = int(
        """6f59 2d67 6f53 6874 746f 2068 6e6b 776f
           2073 6874 2065 6167 6574 202e 6f59 2d67
           6f53 6874 746f 2068 7369 7420 6568 6720
           7461 2e65 5920 676f 532d 746f 6f68 6874
           6920 2073 6874 2065 656b 2079 6e61 2064
           7567 7261 6964 6e61 6f20 2066 6874 2065
           6167 6574 202e 6150 7473 202c 7270 7365
           6e65 2c74 6620 7475 7275 2c65 6120 6c6c
           6120 6572 6f20 656e 6920 206e 6f59 2d67
           6f53 6874 746f 2e68 4820 2065 6e6b 776f
           2073 6877 7265 2065 6874 2065 6c4f 2064
           6e4f 7365 6220 6f72 656b 7420 7268 756f""".replace(' ','').replace('\n',''), 16)

    _X = int(
        """76a3 d17d 5c55 b03e 865f a3e8 2679 90a7
           24ba a24b 0bdd 0cc4 af93 be8d e30b e120
           d553 3c91 bf63 ef92 3b02 edcb 84b7 4438
           3f7d e232 cca6 eb46 d07c ad83 dcaa 317f
           becb c68c a13e 2c40 19e6 a365 3106 7450
           04ae cc0b e1df f0a7 8733 fb0e 7d5c b7c4
           97ca b77b 1331 bf34 7e5f 3a78 47aa 0bc0
           f4bc 6414 6b48 407f ed7b 931d 1697 2d25
           fb4d a5e6 dc07 4ce2 a58d aa8d e762 4247
           cdf2 ebe4 e4df ec6d 5989 aac7 78c8 7559
           d321 3d60 40d4 111c e3a2 acae 19f9 ee15
           3250 9e03 7f69 b252 fdc3 0243 cbbc e9d0""".replace(' ','').replace('\n',''), 16)

    #
    # Test keypair y/Y
    #
    _y = int(
        """7365 6220 6f72 656b 7420 7268 756f 6867
           6f20 2066 6c6f 2c64 6120 646e 7720 6568
           6572 5420 6568 2079 6873 6c61 206c 7262
           6165 206b 6874 6f72 6775 2068 6761 6961
           2e6e 4820 2065 6e6b 776f 2073 6877 7265
           2065 6854 7965 6820 7661 2065 7274 646f
           6520 7261 6874 7327 6620 6569 646c 2c73
           6120 646e 7720 6568 6572 5420 6568 2079
           7473 6c69 206c 7274 6165 2064 6874 6d65
           202c 6e61 2064 6877 2079 6f6e 6f20 656e
           6320 6e61 6220 6865 6c6f 2064 6854 6d65
           6120 2073 6854 7965 7420 6572 6461 0a2e""".replace(' ','').replace('\n',''), 16)

    _Y = int(
        """d04e 156e 554c 37ff d7ab a749 df66 2350
           1e4f f446 6cb1 2be0 5561 7c1a 3687 2237
           36d2 c3fd ce9e e0f9 b277 7435 0849 112a
           a5ae b1f1 2681 1c9c 2f3a 9cb1 3d2f 0c3a
           7e6f a2d3 bf71 baf5 0d83 9171 534f 227e
           fbb2 ce42 27a3 8c25 abdc 5ba7 fc43 0111
           3a2c b206 9c9b 305f aac4 b72b f21f ec71
           578a 9c36 9bca c84e 1a7d cf07 54e3 42f5
           bc8f e491 7441 b882 5443 5e2a baf2 97e9
           3e1e 5796 8672 d45b d7d4 c8ba 1bc3 d314
           889b 5bc3 d3e4 ea33 d4f2 dfdd 34e5 e5a7
           2ff2 4ee4 6316 d475 7dad 0936 6a0b 66b3""".replace(' ','').replace('\n',''), 16)

    #
    # Shared secret: x + Y/y + X
    #
    _xYyX = int(
        """78af af5f 457f 1fdb 832b ebc3 9764 4a33
           038b e9db a10c a2ce 4a07 6f32 7f3a 0ce3
           151d 477b 869e e7ac 4677 5529 2ad8 a77d
           b9bd 87ff bbc3 9955 bcfb 03b1 5838 88c8
           fd03 7834 ff3f 401d 463c 10f8 99aa 6378
           4451 40b7 f838 6a7d 509e 7b9d b19b 677f
           062a 7a1a 4e15 0960 4d7a 0839 ccd5 da61
           73e1 0afd 9eab 6dda 7453 9d60 493c a37f
           a5c9 8cd9 640b 409c d8bb 3be2 bc51 36fd
           42e7 64fc 3f3c 0ddb 8db3 d87a bcf2 e659
           8d2b 101b ef7a 56f5 0ebc 658f 9df1 287d
           a813 5954 3e77 e4a4 cfa7 598a 4152 e4c0""".replace(' ','').replace('\n',''), 16)

    def __init__(self, methodName='runTest'):
        self._x_str = obfs3_dh.int_to_bytes(self._x, 192)
        self._X_str = obfs3_dh.int_to_bytes(self._X, 192)

        self._y_str = obfs3_dh.int_to_bytes(self._y, 192)
        self._Y_str = obfs3_dh.int_to_bytes(self._Y, 192)

        self._xYyX_str = obfs3_dh.int_to_bytes(self._xYyX, 192)

        twisted.trial.unittest.TestCase.__init__(self, methodName)

    def test_odd_key(self):
        dh_x = obfs3_dh.UniformDH(self._x_str)
        self.assertEqual(self._x_str, dh_x.priv_str)
        self.assertEqual(self._X_str, dh_x.get_public())

    def test_even_key(self):
        dh_y = obfs3_dh.UniformDH(self._y_str)
        self.assertEqual(self._y_str, dh_y.priv_str)
        self.assertEqual(self._Y_str, dh_y.get_public())

    def test_exchange(self):
        dh_x = obfs3_dh.UniformDH(self._x_str)
        dh_y = obfs3_dh.UniformDH(self._y_str)
        xY = dh_x.get_secret(dh_y.get_public())
        yX = dh_y.get_secret(dh_x.get_public())
        self.assertEqual(self._xYyX_str,  xY)
        self.assertEqual(self._xYyX_str,  yX)

class testUniformDH_Benchmark(twisted.trial.unittest.TestCase):
    def test_benchmark(self):
        start = time.clock()
        for i in range(0, 1000):
            dh_x = obfs3_dh.UniformDH()
            dh_y = obfs3_dh.UniformDH()
            xY = dh_x.get_secret(dh_y.get_public())
            yX = dh_y.get_secret(dh_x.get_public())
            self.assertEqual(xY, yX)
        end = time.clock()
        taken = (end - start) / 1000 / 2
        log.msg("Generate + Exchange: %f sec" % taken)

if __name__ == '__main__':
    unittest.main()
