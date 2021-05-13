#!/usr/bin/python
# -*- coding: utf-8 -*-

"""
Parts of pyptlib that are useful both to clients and servers.
"""

import os

SUPPORTED_TRANSPORT_VERSIONS = ['1']

def env_has_k(k, v):
    """
    Default validator for :func:`get_env`.

    :returns: str -- The value of the envvar `k` if it is set.
    :raises: :class:`ValueError` if `k` was not found.
    """
    if v is None: raise ValueError('Missing environment variable %s' % k)
    return v

class Config(object):
    """
    pyptlib's configuration.

    :var string stateLocation: Location where application should store state.
    :var list managedTransportVer: List of managed-proxy protocol versions that Tor supports.
    :var list transports: Strings of pluggable transport names that Tor wants us to handle.
    :var bool allTransportsEnabled: True if Tor wants us to spawn all the transports.
    """

    def __init__(self, stateLocation,
                 managedTransportVer=None,
                 transports=None):
        self.stateLocation = stateLocation
        self.managedTransportVer = managedTransportVer or SUPPORTED_TRANSPORT_VERSIONS
        transports = transports or []
        self.allTransportsEnabled = False
        if '*' in transports:
            self.allTransportsEnabled = True
            transports.remove('*')
        self.transports = transports

    def getStateLocation(self):
        """
        :returns: string -- The state location.
        """

        return self.stateLocation

    def getManagedTransportVersions(self):
        """
        :returns: list -- The managed-proxy protocol versions that Tor supports.
        """

        return self.managedTransportVer

    def getAllTransportsEnabled(self):
        """
        Check if Tor wants the application to spawn all its transpotrs.

        :returns: bool -- True if Tor wants the application to spawn all its transports.
        """

        return self.allTransportsEnabled

def get_env(key, validate=env_has_k):
    """
    Get the value of an environment variable.

    :param str key: Environment variable key.
    :param f validate: Function that takes a `var` and a `value`, and returns
        a (maybe transformed) value if it is valid, or throws an exception.
        If the environment does not set `var`, `value` is passed in as `None`.
        The default validator is :func:`env_has_k` which passes any value
        which is set (i.e. not `None`).

    :returns: str -- The value of the envrionment variable.
    :raises: :class:`pyptlib.config.EnvError` if environment variable could not be
            found, or if it did not pass validation.
    """
    try:
        return validate(key, os.getenv(key))
    except ProxyError:
        raise
    except Exception, e:
        raise EnvError("error parsing env-var: %s: %s" % (key, e), e)

class ProxyError(Exception):
    """
    Thrown when the proxy specifier is incomplete or corrupted.
    """
    def __init__(self, message=None, cause=None):
        self.message = message
        self.cause = cause

    def __str__(self):
        return self.message or self.cause.message

class EnvError(Exception):
    """
    Thrown when the environment is incomplete or corrupted.
    """
    def __init__(self, message=None, cause=None):
        self.message = message
        self.cause = cause

    def __str__(self):
        return self.message or self.cause.message

def checkClientMode():
    """
    Read the environment and return true if we are supposed to be a
    client. Return false if we are supposed to be a server.

    :raises: :class:`pyptlib.config.EnvError` if the environment was not
             properly set up
    """
    if 'TOR_PT_CLIENT_TRANSPORTS' in os.environ: return True
    if 'TOR_PT_SERVER_TRANSPORTS' in os.environ: return False
    raise EnvError('neither TOR_PT_{SERVER,CLIENT}_TRANSPORTS set')
