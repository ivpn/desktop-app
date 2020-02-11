import unittest

from Crypto.Cipher import AES
from Crypto.Util import Counter

import obfsproxy.common.aes as aes
import twisted.trial.unittest

class testAES_CTR_128_NIST(twisted.trial.unittest.TestCase):
    def _helper_test_vector(self, input_block, output_block, plaintext, ciphertext):
        self.assertEqual(long(input_block.encode('hex'), 16), self.ctr.next_value())

        ct = self.cipher.encrypt(plaintext)
        self.assertEqual(ct, ciphertext)

        # XXX how do we extract the keystream out of the AES object?

    def test_nist(self):
        # Prepare the cipher
        key = "\x2b\x7e\x15\x16\x28\xae\xd2\xa6\xab\xf7\x15\x88\x09\xcf\x4f\x3c"
        iv = "\xf0\xf1\xf2\xf3\xf4\xf5\xf6\xf7\xf8\xf9\xfa\xfb\xfc\xfd\xfe\xff"

        self.ctr = Counter.new(128, initial_value=long(iv.encode('hex'), 16))
        self.cipher = AES.new(key, AES.MODE_CTR, counter=self.ctr)

        input_block = "\xf0\xf1\xf2\xf3\xf4\xf5\xf6\xf7\xf8\xf9\xfa\xfb\xfc\xfd\xfe\xff"
        output_block = "\xec\x8c\xdf\x73\x98\x60\x7c\xb0\xf2\xd2\x16\x75\xea\x9e\xa1\xe4"
        plaintext = "\x6b\xc1\xbe\xe2\x2e\x40\x9f\x96\xe9\x3d\x7e\x11\x73\x93\x17\x2a"
        ciphertext = "\x87\x4d\x61\x91\xb6\x20\xe3\x26\x1b\xef\x68\x64\x99\x0d\xb6\xce"

        self._helper_test_vector(input_block, output_block, plaintext, ciphertext)

        input_block = "\xf0\xf1\xf2\xf3\xf4\xf5\xf6\xf7\xf8\xf9\xfa\xfb\xfc\xfd\xff\x00"
        output_block = "\x36\x2b\x7c\x3c\x67\x73\x51\x63\x18\xa0\x77\xd7\xfc\x50\x73\xae"
        plaintext = "\xae\x2d\x8a\x57\x1e\x03\xac\x9c\x9e\xb7\x6f\xac\x45\xaf\x8e\x51"
        ciphertext = "\x98\x06\xf6\x6b\x79\x70\xfd\xff\x86\x17\x18\x7b\xb9\xff\xfd\xff"

        self._helper_test_vector(input_block, output_block, plaintext, ciphertext)

        input_block = "\xf0\xf1\xf2\xf3\xf4\xf5\xf6\xf7\xf8\xf9\xfa\xfb\xfc\xfd\xff\x01"
        output_block = "\x6a\x2c\xc3\x78\x78\x89\x37\x4f\xbe\xb4\xc8\x1b\x17\xba\x6c\x44"
        plaintext = "\x30\xc8\x1c\x46\xa3\x5c\xe4\x11\xe5\xfb\xc1\x19\x1a\x0a\x52\xef"
        ciphertext = "\x5a\xe4\xdf\x3e\xdb\xd5\xd3\x5e\x5b\x4f\x09\x02\x0d\xb0\x3e\xab"

        self._helper_test_vector(input_block, output_block, plaintext, ciphertext)

        input_block = "\xf0\xf1\xf2\xf3\xf4\xf5\xf6\xf7\xf8\xf9\xfa\xfb\xfc\xfd\xff\x02"
        output_block = "\xe8\x9c\x39\x9f\xf0\xf1\x98\xc6\xd4\x0a\x31\xdb\x15\x6c\xab\xfe"
        plaintext = "\xf6\x9f\x24\x45\xdf\x4f\x9b\x17\xad\x2b\x41\x7b\xe6\x6c\x37\x10"
        ciphertext = "\x1e\x03\x1d\xda\x2f\xbe\x03\xd1\x79\x21\x70\xa0\xf3\x00\x9c\xee"

        self._helper_test_vector(input_block, output_block, plaintext, ciphertext)

class testAES_CTR_128_simple(twisted.trial.unittest.TestCase):
    def test_encrypt_decrypt_small_ASCII(self):
        """
        Validate that decryption and encryption work as intended on a small ASCII string.
        """
        self.key = "\xe3\xb0\xc4\x42\x98\xfc\x1c\x14\x9a\xfb\xf4\xc8\x99\x6f\xb9\x24"
        self.iv = "\x27\xae\x41\xe4\x64\x9b\x93\x4c\xa4\x95\x99\x1b\x78\x52\xb8\x55"

        test_string = "This unittest kills fascists."

        cipher1 = aes.AES_CTR_128(self.key, self.iv)
        cipher2 = aes.AES_CTR_128(self.key, self.iv)

        ct = cipher1.crypt(test_string)
        pt = cipher2.crypt(ct)

        self.assertEqual(test_string, pt)


if __name__ == '__main__':
    unittest.main()

