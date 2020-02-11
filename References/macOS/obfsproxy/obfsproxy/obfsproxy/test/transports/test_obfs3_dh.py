import unittest
import twisted.trial.unittest

import obfsproxy.transports.obfs3_dh as obfs3_dh

class test_uniform_dh(twisted.trial.unittest.TestCase):
    def test_uniform_dh(self):
        alice = obfs3_dh.UniformDH()
        bob = obfs3_dh.UniformDH()

        alice_pub = alice.get_public()
        bob_pub = bob.get_public()

        alice_secret = alice.get_secret(bob_pub)
        bob_secret = bob.get_secret(alice_pub)

        self.assertEqual(alice_secret, bob_secret)

if __name__ == '__main__':
    unittest.main()

