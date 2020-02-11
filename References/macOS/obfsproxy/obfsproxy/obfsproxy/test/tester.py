#!/usr/bin/python

"""@package tester.py.in
Integration tests for obfsproxy.

The obfsproxy binary is assumed to exist in the current working
directory, and you need to have Python 2.6 or better (but not 3).
You need to be able to make connections to arbitrary high-numbered
TCP ports on the loopback interface.
"""

import difflib
import errno
import multiprocessing
import Queue
import re
import signal
import socket
import struct
import subprocess
import time
import traceback
import unittest
import sys,os
import tempfile
import shutil

def diff(label, expected, received):
    """
    Helper: generate unified-format diffs between two named strings.
    Pythonic escaped-string syntax is used for unprintable characters.
    """
    if expected == received:
        return ""
    else:
        return (label + "\n"
                + "\n".join(s.encode("string_escape")
                            for s in
                            difflib.unified_diff(expected.split("\n"),
                                                 received.split("\n"),
                                                 "expected", "received",
                                                 lineterm=""))
                + "\n")

class Obfsproxy(subprocess.Popen):
    """
    Helper: Run obfsproxy instances and confirm that they have
    completed without any errors.
    """
    def __init__(self, *args, **kwargs):
        """Spawns obfsproxy with 'args'"""
        argv = ["bin/obfsproxy", "--no-log"]
        if len(args) == 1 and (isinstance(args[0], list) or
                               isinstance(args[0], tuple)):
            argv.extend(args[0])
        else:
            argv.extend(args)

        subprocess.Popen.__init__(self, argv,
                                  stdin=open("/dev/null", "r"),
                                  stdout=subprocess.PIPE,
                                  stderr=subprocess.PIPE,
                                  **kwargs)

    severe_error_re = re.compile(r"\[(?:warn|err(?:or)?)\]")

    def check_completion(self, label, force_stderr):
        """
        Checks the output and exit status of obfsproxy to see if
        everything went fine.

        Returns an empty string if the test was good, otherwise it
        returns a report that should be printed to the user.
        """
        if self.poll() is None:
            self.send_signal(signal.SIGINT)

        (out, err) = self.communicate()

        report = ""
        def indent(s):
            return "| " + "\n| ".join(s.strip().split("\n"))

        # exit status should be zero
        if self.returncode > 0:
            report += label + " exit code: %d\n" % self.returncode
        elif self.returncode < 0:
            report += label + " killed: signal %d\n" % -self.returncode

        # there should be nothing on stdout
        if out != "":
            report += label + " stdout:\n%s\n" % indent(out)

        # there will be debugging messages on stderr, but there should be
        # no [warn], [err], or [error] messages.
        if force_stderr or self.severe_error_re.search(err):
            report += label + " stderr:\n%s\n" % indent(err)

        return report

    def stop(self):
        """Terminates obfsproxy."""
        if self.poll() is None:
            self.terminate()

def connect_with_retry(addr):
    """
    Helper: Repeatedly try to connect to the specified server socket
    until either it succeeds or one full second has elapsed.  (Surely
    there is a better way to do this?)
    """

    retry = 0
    while True:
        try:
            return socket.create_connection(addr)
        except socket.error, e:
            if e.errno != errno.ECONNREFUSED: raise
            if retry == 20: raise
            retry += 1
            time.sleep(0.05)

SOCKET_TIMEOUT = 2.0

class ReadWorker(object):
    """
    Helper: In a separate process (to avoid deadlock), listen on a
    specified socket.  The first time something connects to that socket,
    read all available data, stick it in a string, and post the string
    to the output queue.  Then close both sockets and exit.
    """

    @staticmethod
    def work(address, oq):
        listener = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        listener.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        listener.bind(address)
        listener.listen(1)
        (conn, remote) = listener.accept()
        listener.close()
        conn.settimeout(SOCKET_TIMEOUT)
        data = ""
        try:
            while True:
                chunk = conn.recv(4096)
                if chunk == "": break
                data += chunk
        except socket.timeout:
            pass
        except Exception, e:
            data += "|RECV ERROR: " + e
        conn.close()
        oq.put(data)

    def __init__(self, address):
        self.oq = multiprocessing.Queue()
        self.worker = multiprocessing.Process(target=self.work,
                                              args=(address, self.oq))
        self.worker.start()

    def get(self):
        """
        Get a chunk of data from the ReadWorker's queue.
        """
        rv = self.oq.get(timeout=SOCKET_TIMEOUT+0.1)
        self.worker.join()
        return rv

    def stop(self):
        if self.worker.is_alive(): self.worker.terminate()

# Right now this is a direct translation of the former int_test.sh
# (except that I have fleshed out the SOCKS test a bit).
# It will be made more general and parametric Real Soon.

ENTRY_PORT  = 4999
SERVER_PORT = 5000
EXIT_PORT   = 5001

#
# Test base classes.  They do _not_ inherit from unittest.TestCase
# so that they are not scanned directly for test functions (some of
# them do provide test functions, but not in a usable state without
# further code from subclasses).
#

class DirectTest(object):
    def setUp(self):
        self.output_reader = ReadWorker(("127.0.0.1", EXIT_PORT))
        self.obfs_server = Obfsproxy(self.server_args)
        time.sleep(0.1)
        self.obfs_client = Obfsproxy(self.client_args)
        self.input_chan = connect_with_retry(("127.0.0.1", ENTRY_PORT))
        self.input_chan.settimeout(SOCKET_TIMEOUT)

    def tearDown(self):
        self.obfs_client.stop()
        self.obfs_server.stop()
        self.output_reader.stop()
        self.input_chan.close()

    def test_direct_transfer(self):
        # Open a server and a simple client (in the same process) and
        # transfer a file.  Then check whether the output is the same
        # as the input.
        self.input_chan.sendall(TEST_FILE)
        time.sleep(2)
        try:
            output = self.output_reader.get()
        except Queue.Empty:
            output = ""

        self.input_chan.close()

        report = diff("errors in transfer:", TEST_FILE, output)

        report += self.obfs_client.check_completion("obfsproxy client (%s)" % self.transport, report!="")
        report += self.obfs_server.check_completion("obfsproxy server (%s)" % self.transport, report!="")

        if report != "":
            self.fail("\n" + report)

#
# Concrete test classes specialize the above base classes for each protocol.
#

class DirectDummy(DirectTest, unittest.TestCase):
    transport = "dummy"
    server_args = ("dummy", "server",
                   "127.0.0.1:%d" % SERVER_PORT,
                   "--dest=127.0.0.1:%d" % EXIT_PORT)
    client_args = ("dummy", "client",
                   "127.0.0.1:%d" % ENTRY_PORT,
                   "--dest=127.0.0.1:%d" % SERVER_PORT)

class DirectObfs2(DirectTest, unittest.TestCase):
    transport = "obfs2"
    server_args = ("obfs2", "server",
                   "127.0.0.1:%d" % SERVER_PORT,
                   "--dest=127.0.0.1:%d" % EXIT_PORT)
    client_args = ("obfs2", "client",
                   "127.0.0.1:%d" % ENTRY_PORT,
                   "--dest=127.0.0.1:%d" % SERVER_PORT)

class DirectObfs2_ss(DirectTest, unittest.TestCase):
    transport = "obfs2"
    server_args = ("obfs2", "server",
                   "127.0.0.1:%d" % SERVER_PORT,
                   "--shared-secret=test",
                   "--ss-hash-iterations=50",
                   "--dest=127.0.0.1:%d" % EXIT_PORT)
    client_args = ("obfs2", "client",
                   "127.0.0.1:%d" % ENTRY_PORT,
                   "--shared-secret=test",
                   "--ss-hash-iterations=50",
                   "--dest=127.0.0.1:%d" % SERVER_PORT)

class DirectB64(DirectTest, unittest.TestCase):
    transport = "b64"
    server_args = ("b64", "server",
                   "127.0.0.1:%d" % SERVER_PORT,
                   "--dest=127.0.0.1:%d" % EXIT_PORT)
    client_args = ("b64", "client",
                   "127.0.0.1:%d" % ENTRY_PORT,
                   "--dest=127.0.0.1:%d" % SERVER_PORT)

class DirectObfs3(DirectTest, unittest.TestCase):
    transport = "obfs3"
    server_args = ("obfs3", "server",
                   "127.0.0.1:%d" % SERVER_PORT,
                   "--dest=127.0.0.1:%d" % EXIT_PORT)
    client_args = ("obfs3", "client",
                   "127.0.0.1:%d" % ENTRY_PORT,
                   "--dest=127.0.0.1:%d" % SERVER_PORT)

class DirectScrambleSuit(DirectTest, unittest.TestCase):
    transport = "scramblesuit"

    def setUp(self):
        # First, we need to create data directories for ScrambleSuit.  It uses
        # them to store persistent information such as session tickets and the
        # server's long-term keys.
        self.tmpdir_srv = tempfile.mkdtemp(prefix="server")
        self.tmpdir_cli = tempfile.mkdtemp(prefix="client")

        self.server_args = ("--data-dir=%s" % self.tmpdir_srv,
                            "scramblesuit", "server",
                            "127.0.0.1:%d" % SERVER_PORT,
                            "--password=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
                            "--dest=127.0.0.1:%d" % EXIT_PORT)
        self.client_args = ("--data-dir=%s" % self.tmpdir_cli,
                            "scramblesuit", "client",
                            "127.0.0.1:%d" % ENTRY_PORT,
                            "--password=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
                            "--dest=127.0.0.1:%d" % SERVER_PORT)

        # Now, the remaining setup steps can be done.
        super(DirectScrambleSuit, self).setUp()

    def tearDown(self):
        # First, let the parent class shut down the test.
        super(DirectScrambleSuit, self).tearDown()

        # Now, we can clean up after ourselves.
        shutil.rmtree(self.tmpdir_srv)
        shutil.rmtree(self.tmpdir_cli)


TEST_FILE = """\
THIS IS A TEST FILE. IT'S USED BY THE INTEGRATION TESTS.
THIS IS A TEST FILE. IT'S USED BY THE INTEGRATION TESTS.
THIS IS A TEST FILE. IT'S USED BY THE INTEGRATION TESTS.
THIS IS A TEST FILE. IT'S USED BY THE INTEGRATION TESTS.

"Can entropy ever be reversed?"
"THERE IS AS YET INSUFFICIENT DATA FOR A MEANINGFUL ANSWER."
"Can entropy ever be reversed?"
"THERE IS AS YET INSUFFICIENT DATA FOR A MEANINGFUL ANSWER."
"Can entropy ever be reversed?"
"THERE IS AS YET INSUFFICIENT DATA FOR A MEANINGFUL ANSWER."
"Can entropy ever be reversed?"
"THERE IS AS YET INSUFFICIENT DATA FOR A MEANINGFUL ANSWER."
"Can entropy ever be reversed?"
"THERE IS AS YET INSUFFICIENT DATA FOR A MEANINGFUL ANSWER."
"Can entropy ever be reversed?"
"THERE IS AS YET INSUFFICIENT DATA FOR A MEANINGFUL ANSWER."
"Can entropy ever be reversed?"
"THERE IS AS YET INSUFFICIENT DATA FOR A MEANINGFUL ANSWER."
"Can entropy ever be reversed?"
"THERE IS AS YET INSUFFICIENT DATA FOR A MEANINGFUL ANSWER."

    In obfuscatory age geeky warfare did I wage
      For hiding bits from nasty censors' sight
    I was hacker to my set in that dim dark age of net
      And I hacked from noon till three or four at night

    Then a rival from Helsinki said my protocol was dinky
      So I flamed him with a condescending laugh,
    Saying his designs for stego might as well be made of lego
      And that my bikeshed was prettier by half.

    But Claude Shannon saw my shame. From his noiseless channel came
       A message sent with not a wasted byte
    "There are nine and sixty ways to disguise communiques
       And RATHER MORE THAN ONE OF THEM IS RIGHT"

		    (apologies to Rudyard Kipling.)
"""

if __name__ == '__main__':
    unittest.main()
