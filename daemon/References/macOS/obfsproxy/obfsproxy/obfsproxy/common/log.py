"""obfsproxy logging code"""
import logging
import sys

from twisted.python import log

def get_obfslogger():
    """ Return the current ObfsLogger instance """
    return OBFSLOGGER


class ObfsLogger(object):
    """
    Maintain state of logging options specified with command line arguments

    Attributes:
    safe_logging: Boolean value indicating if we should scrub addresses
                  before logging
    obfslogger: Our logging instance
    """

    def __init__(self):

        self.safe_logging = True

        observer = log.PythonLoggingObserver('obfslogger')
        observer.start()

        # Create the default log handler that logs to stdout.
        self.obfslogger = logging.getLogger('obfslogger')
        self.default_handler = logging.StreamHandler(sys.stdout)
        self.set_formatter(self.default_handler)
        self.obfslogger.addHandler(self.default_handler)
        self.obfslogger.propagate = False

    def set_formatter(self, handler):
        """Given a log handler, plug our custom formatter to it."""

        formatter = logging.Formatter("%(asctime)s [%(levelname)s] %(message)s")
        handler.setFormatter(formatter)

    def set_log_file(self, filename):
        """Set up our logger so that it starts logging to file in 'filename' instead."""

        # remove the default handler, and add the FileHandler:
        self.obfslogger.removeHandler(self.default_handler)

        log_handler = logging.FileHandler(filename)
        self.set_formatter(log_handler)

        self.obfslogger.addHandler(log_handler)


    def set_log_severity(self, sev_string):
        """Update our minimum logging severity to 'sev_string'."""

        # Turn it into a numeric level that logging understands first.
        numeric_level = getattr(logging, sev_string.upper(), None)
        self.obfslogger.setLevel(numeric_level)


    def disable_logs(self):
        """Disable all logging."""

        logging.disable(logging.CRITICAL)


    def set_no_safe_logging(self):
        """ Disable safe_logging """

        self.safe_logging = False


    def safe_addr_str(self, address):
        """
        Unless safe_logging is False, we return '[scrubbed]' instead
        of the address parameter. If safe_logging is false, then we
        return the address itself.
        """

        if self.safe_logging:
            return '[scrubbed]'
        else:
            return address

    def debug(self, msg, *args, **kwargs):
        """ Class wrapper around debug logging method """

        self.obfslogger.debug(msg, *args, **kwargs)

    def warning(self, msg, *args, **kwargs):
        """ Class wrapper around warning logging method """

        self.obfslogger.warning(msg, *args, **kwargs)

    def info(self, msg, *args, **kwargs):
        """ Class wrapper around info logging method """

        self.obfslogger.info(msg, *args, **kwargs)

    def error(self, msg, *args, **kwargs):
        """ Class wrapper around error logging method """

        self.obfslogger.error(msg, *args, **kwargs)

    def critical(self, msg, *args, **kwargs):
        """ Class wrapper around critical logging method """

        self.obfslogger.critical(msg, *args, **kwargs)

    def exception(self, msg, *args, **kwargs):
        """ Class wrapper around exception logging method """

        self.obfslogger.exception(msg, *args, **kwargs)

""" Global variable that will track our Obfslogger instance """
OBFSLOGGER = ObfsLogger()
