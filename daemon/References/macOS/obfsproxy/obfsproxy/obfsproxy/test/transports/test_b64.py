import unittest
import twisted.trial.unittest

import obfsproxy.transports.b64 as b64

class test_b64_splitting(twisted.trial.unittest.TestCase):
    def _helper_splitter(self, string, expected_chunks):
        chunks = b64._get_b64_chunks_from_str(string)
        self.assertEqual(chunks, expected_chunks)

    def test_1(self):
        string = "on==the==left==hand==side=="
        expected = ["on==", "the==", "left==", "hand==", "side=="]
        self._helper_splitter(string, expected)

    def test_2(self):
        string = "on=the=left=hand=side="
        expected = ["on=", "the=", "left=", "hand=", "side="]
        self._helper_splitter(string, expected)

    def test_3(self):
        string = "on==the=left==hand=side=="
        expected = ["on==", "the=", "left==", "hand=", "side=="]
        self._helper_splitter(string, expected)

    def test_4(self):
        string = "on==the==left=hand=side"
        expected = ["on==", "the==", "left=", "hand=", "side"]
        self._helper_splitter(string, expected)

    def test_5(self):
        string = "onthelefthandside=="
        expected = ["onthelefthandside=="]
        self._helper_splitter(string, expected)

    def test_6(self):
        string = "onthelefthandside"
        expected = ["onthelefthandside"]
        self._helper_splitter(string, expected)

    def test_7(self):
        string = "onthelefthandside="
        expected = ["onthelefthandside="]
        self._helper_splitter(string, expected)

    def test_8(self):
        string = "side=="
        expected = ["side=="]
        self._helper_splitter(string, expected)

    def test_9(self):
        string = "side="
        expected = ["side="]
        self._helper_splitter(string, expected)

    def test_10(self):
        string = "side"
        expected = ["side"]
        self._helper_splitter(string, expected)

if __name__ == '__main__':
    unittest.main()

