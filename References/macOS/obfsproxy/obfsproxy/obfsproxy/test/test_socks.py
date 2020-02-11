import obfsproxy.network.socks as socks

import twisted.trial.unittest

class test_SOCKS(twisted.trial.unittest.TestCase):
    def test_socks_args_splitting(self):
        socks_args = socks._split_socks_args("monday=blue;tuesday=grey;wednesday=too;thursday=don\\;tcareabout\\\\you;friday=i\\;minlove")
        self.assertListEqual(socks_args, ["monday=blue", "tuesday=grey", "wednesday=too", "thursday=don;tcareabout\\you", "friday=i;minlove"])

        socks_args = socks._split_socks_args("monday=blue")
        self.assertListEqual(socks_args, ["monday=blue"])

        socks_args = socks._split_socks_args("monday=;tuesday=grey")
        self.assertListEqual(socks_args, ["monday=", "tuesday=grey"])

        socks_args = socks._split_socks_args("\\;=\\;;\\\\=\\;")
        self.assertListEqual(socks_args, [";=;", "\\=;"])

