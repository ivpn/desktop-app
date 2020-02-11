"""Common tasks for managing child processes.

To have child processes actually be managed by this module, you should use
the Popen() here rather than subprocess.Popen() directly.

Some parts do not yet work fully on windows (sending/trapping signals).
"""

import atexit
import inspect
import os
import signal
import subprocess
import sys
import time

mswindows = (sys.platform == "win32")
if mswindows:
    """
    Set KILL_CHILDREN_ON_DEATH=1 in the environment to automatically kill all
    descendents when this process dies.
    """
    # TODO(infinity0): write a test for this, similar to test_killall_kill
    # Note: setting this to True defeats the point of some of the tests, so
    # keep the default value as False. Perhaps we could make that work better.
    _kill_children_on_death = bool(os.getenv("KILL_CHILDREN_ON_DEATH", 0))

    from ctypes import byref, windll, WinError
    from ctypes.wintypes import DWORD
    import win32api, win32con, win32job, win32process

_CHILD_PROCS = []
# TODO(infinity0): add functionality to detect when any child dies, and
# offer different response strategies for them (e.g. restart the child? or die
# and kill the other children too).

SINK = object()

# get default args from subprocess.Popen to use in subproc.Popen
a = inspect.getargspec(subprocess.Popen.__init__)
_Popen_defaults = zip(a.args[-len(a.defaults):],a.defaults); del a
if mswindows:
    # required for os.kill() to work
    _Popen_creationflags = subprocess.CREATE_NEW_PROCESS_GROUP

    if _kill_children_on_death:
        _chJob = win32job.CreateJobObject(None, "")
        if not _chJob:
            raise WinError()

        chJeli = win32job.QueryInformationJobObject(
            _chJob, win32job.JobObjectExtendedLimitInformation)
        # JOB_OBJECT_LIMIT_BREAKAWAY_OK allows children to assign grandchildren
        # to their own jobs
        chJeli['BasicLimitInformation']['LimitFlags'] |= (
            win32job.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE |
            win32job.JOB_OBJECT_LIMIT_BREAKAWAY_OK)

        if win32job.SetInformationJobObject(
            _chJob, win32job.JobObjectExtendedLimitInformation, chJeli) == 0:
            raise WinError()
        del chJeli

        # If we already belong to a JobObject, our children are auto-assigned
        # to that and AssignProcessToJobObject(ch, _chJob) fails. This flag
        # prevents this auto-assignment, as long as the parent JobObject has
        # JOB_OBJECT_LIMIT_BREAKAWAY_OK set on it as well.
        _Popen_creationflags |= win32process.CREATE_BREAKAWAY_FROM_JOB

    tmp = dict(_Popen_defaults)
    tmp['creationflags'] |= _Popen_creationflags
    _Popen_defaults = tmp.items()
    del tmp, _Popen_creationflags


class Popen(subprocess.Popen):
    """Wrapper for subprocess.Popen that tracks every child process.

    See the subprocess module for documentation.

    On windows, you are recommended to use the creationflagsmerge param so as
    not to interfere with the required flags that we set in this module.

    Additionally, you may use subproc.SINK as the value for either of the
    stdout, stderr arguments to tell subprocess to discard anything written
    to those channels.
    """

    def __init__(self, *args, **kwargs):
        kwargs = dict(_Popen_defaults + kwargs.items())
        if 'creationflagsmerge' in kwargs:
            kwargs['creationflags'] = (
                kwargs.get('creationflags', 0) | kwargs['creationflagsmerge'])
            del kwargs['creationflagsmerge']
        for f in ['stdout', 'stderr']:
            if kwargs[f] is SINK:
                kwargs[f] = create_sink()
        # super() does some magic that makes **kwargs not work, so just call
        # our super-constructor directly
        subprocess.Popen.__init__(self, *args, **kwargs)
        _CHILD_PROCS.append(self)

        if mswindows and _kill_children_on_death:
            handle = windll.kernel32.OpenProcess(
                win32con.SYNCHRONIZE | win32con.PROCESS_SET_QUOTA | win32con.PROCESS_TERMINATE, 0, self.pid)
            if win32job.AssignProcessToJobObject(_chJob, handle) == 0:
                raise WinError()

    # TODO(infinity0): perhaps replace Popen.std* with wrapped file objects
    # that don't buffer readlines() et. al. Currently one must avoid these and
    # use while/readline(); see man page for "python -u" for more details.

def create_sink():
    return open(os.devnull, "w", 0)


if mswindows:
    # adapted from http://www.madebuild.org/blog/?p=30

    def proc_is_alive(pid):
        """Check if a pid is still running."""
        handle = windll.kernel32.OpenProcess(
            win32con.SYNCHRONIZE | win32con.PROCESS_QUERY_INFORMATION, 0, pid)
        if handle == 0:
            return False

        # If the process exited recently, a pid may still exist for the handle.
        # So, check if we can get the exit code.
        exit_code = DWORD()
        rval = windll.kernel32.GetExitCodeProcess(handle, byref(exit_code))
        windll.kernel32.CloseHandle(handle)
        if rval == 0: # GetExitCodeProcess failure
            raise WinError()
        return exit_code.value == win32con.STILL_ACTIVE

else:
    # adapted from http://stackoverflow.com/questions/568271/check-if-pid-is-not-in-use-in-python
    import errno

    def proc_is_alive(pid):
        """Check if a pid is still running."""
        try:
            os.kill(pid, 0)
        except OSError as e:
            if e.errno == errno.EPERM:
                return True
            if e.errno == errno.ESRCH:
                return False
            raise # something else went wrong
        else:
            return True


class SignalHandlers(object):

    def __init__(self):
        self.handlers = {}
        self.received = 0

    def attach_override_unix(self, signum):
        if signal.signal(signum, self.handle) != self.handle:
            self.handlers.clear()

    def handle(self, signum=0, sframe=None):
        self.received += 1

        # code snippet adapted from atexit._run_exitfuncs
        exc_info = None
        for i in xrange(self.received).__reversed__():
            for handler in self.handlers.get(i, []).__reversed__():
                try:
                    handler(signum, sframe)
                except SystemExit:
                    exc_info = sys.exc_info()
                except:
                    import traceback
                    print >> sys.stderr, "Error in SignalHandler.handle:"
                    traceback.print_exc()
                    exc_info = sys.exc_info()

        if exc_info is not None:
            raise exc_info[0], exc_info[1], exc_info[2]

    def register(self, handler, ignoreNum):
        self.handlers.setdefault(ignoreNum, []).append(handler)


_SIGINT_HANDLERS = SignalHandlers()
def trap_sigint(handler, ignoreNum=0):
    """Register a handler for an INT signal (Unix).

    Note: this currently has no effect on windows.

    Successive handlers are cumulative. On Unix, they override any previous
    handlers registered with signal.signal().

    Args:
        handler: a signal handler; see signal.signal() for details
        ignoreNum: number of signals to ignore before activating the handler,
            which will be run on all subsequent signals.
    """
    handlers = _SIGINT_HANDLERS
    handlers.attach_override_unix(signal.SIGINT)
    handlers.register(handler, ignoreNum)


_isTerminating = False
def killall(cleanup=lambda:None, wait_s=16):
    """Attempt to gracefully terminate all child processes.

    All children are told to terminate gracefully. A waiting period is then
    applied, after which all children are killed forcefully. If all children
    terminate before this waiting period is over, the function exits early.

    Args:
        cleanup: Run after all children are dead. For example, if your program
                does not automatically terminate after this, you can use this
                to signal that it should exit. In particular, Twisted
                applications ought to use this to call reactor.stop().
        wait_s: Time in seconds to wait before trying to kill children.
    """
    # TODO(infinity0): log this somewhere, maybe
    global _isTerminating, _CHILD_PROCS
    if _isTerminating: return
    _isTerminating = True
    # terminate all
    for proc in _CHILD_PROCS:
        if proc.poll() is None:
            proc.terminate()
    # wait and make sure they're dead
    for i in xrange(wait_s):
        _CHILD_PROCS = [proc for proc in _CHILD_PROCS
                        if proc.poll() is None]
        if not _CHILD_PROCS: break
        time.sleep(1)
    # if still existing, kill them
    for proc in _CHILD_PROCS:
        if proc.poll() is None:
            proc.kill()
    time.sleep(0.5)
    # reap any zombies
    for proc in _CHILD_PROCS:
        proc.poll()
    cleanup()

def auto_killall(ignoreNumSigInts=0, *args, **kwargs):
    """Automatically terminate all child processes on exit.

    Args:
        ignoreNumSigInts: this number of INT signals will be ignored before
            attempting termination. This will be attempted unconditionally in
            all other cases, such as on normal exit, or on a TERM signal.
        *args, **kwargs: See killall().
    """
    killall_handler = lambda signum, sframe: killall(*args, **kwargs)
    trap_sigint(killall_handler, ignoreNumSigInts)
    signal.signal(signal.SIGTERM, killall_handler)
    atexit.register(killall, *args, **kwargs)
