"""heartbeat code"""

import datetime
import socket # for socket.inet_pton()

import obfsproxy.common.log as logging

log = logging.get_obfslogger()

def get_integer_from_ip_str(ip_str):
    """
    Given an IP address in string format in <b>ip_str</b>, return its
    integer representation.

    Throws ValueError if the IP address string was invalid.
    """
    try:
        return socket.inet_pton(socket.AF_INET, ip_str)
    except socket.error:
        pass

    try:
        return socket.inet_pton(socket.AF_INET6, ip_str)
    except socket.error:
        pass

    # Down here, both inet_pton()s failed.
    raise ValueError("Invalid IP address string")

class Heartbeat(object):
    """
    Represents obfsproxy's heartbeat.

    It keeps stats on a number of things that the obfsproxy operator
    might be interested in, and every now and then it reports them in
    the logs.

    'unique_ips': A Python set that contains unique IPs (in integer
    form) that have connected to obfsproxy.
    """

    def __init__(self):
        self.n_connections = 0
        self.started = datetime.datetime.now()
        self.last_reset = self.started
        self.unique_ips = set()

    def register_connection(self, ip_str):
        """Register a new connection."""
        self.n_connections += 1
        self._register_ip(ip_str)

    def _register_ip(self, ip_str):
        """
        See if 'ip_str' has connected to obfsproxy before. If not, add
        it to the list of unique IPs.
        """
        ip = get_integer_from_ip_str(ip_str)
        if ip not in self.unique_ips:
            self.unique_ips.add(ip)

    def reset_stats(self):
        """Reset stats."""

        self.n_connections = 0
        self.unique_ips = set()
        self.last_reset = datetime.datetime.now()

    def say_uptime(self):
        """Log uptime information."""

        now = datetime.datetime.now()
        delta = now - self.started

        uptime_days = delta.days
        uptime_hours = round(float(delta.seconds)/3600)
        uptime_minutes = round(float(delta.seconds)/60)%60

        if uptime_days:
            log.info("Heartbeat: obfsproxy's uptime is %d day(s), %d hour(s) and %d minute(s)." % \
                         (uptime_days, uptime_hours, uptime_minutes))
        else:
            log.info("Heartbeat: obfsproxy's uptime is %d hour(s) and %d minute(s)." % \
                         (uptime_hours, uptime_minutes))

    def say_stats(self):
        """Log connection stats."""

        now = datetime.datetime.now()
        reset_delta = now - self.last_reset

        log.info("Heartbeat: During the last %d hour(s) we saw %d connection(s)" \
                 " from %d unique address(es)." % \
                     (round(float(reset_delta.seconds/3600)) + reset_delta.days*24, self.n_connections,
                      len(self.unique_ips)))

        # Reset stats every 24 hours.
        if (reset_delta.days > 0):
            log.debug("Resetting heartbeat.")
            self.reset_stats()

    def talk(self):
        """Do a heartbeat."""

        self.say_uptime()
        self.say_stats()

# A heartbeat singleton.
heartbeat = Heartbeat()
