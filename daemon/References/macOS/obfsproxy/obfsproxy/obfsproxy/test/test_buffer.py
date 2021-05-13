import unittest

import obfsproxy.network.buffer as obfs_buf
import twisted.trial.unittest

class testBuffer(twisted.trial.unittest.TestCase):
    def setUp(self):
        self.test_string = "No pop no style, I strictly roots."
        self.buf = obfs_buf.Buffer(self.test_string)

    def test_totalread(self):
        tmp = self.buf.read(-1)
        self.assertEqual(tmp, self.test_string)

    def test_byte_by_byte(self):
        """Read one byte at a time."""
        for i in xrange(len(self.test_string)):
            self.assertEqual(self.buf.read(1), self.test_string[i])

    def test_bigread(self):
        self.assertEqual(self.buf.read(666), self.test_string)

    def test_peek(self):
        tmp = self.buf.peek(-1)
        self.assertEqual(tmp, self.test_string)
        self.assertEqual(self.buf.read(-1), self.test_string)

    def test_drain(self):
        tmp = self.buf.drain(-1) # drain everything
        self.assertIsNone(tmp) # check non-existent retval
        self.assertEqual(self.buf.read(-1), '') # it should be empty.
        self.assertEqual(len(self.buf), 0)

    def test_drain2(self):
        tmp = self.buf.drain(len(self.test_string)-1) # drain everything but a byte
        self.assertIsNone(tmp) # check non-existent retval
        self.assertEqual(self.buf.peek(-1), '.') # peek at last character
        self.assertEqual(len(self.buf), 1) # length must be 1


if __name__ == '__main__':
    unittest.main()


