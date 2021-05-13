#!/usr/bin/python
# -*- coding: utf-8 -*-

from twisted.internet import reactor, error

import obfsproxy.network.launch_transport as launch_transport
import obfsproxy.network.network as network
import obfsproxy.transports.transports as transports
import obfsproxy.transports.base as base
import obfsproxy.common.log as logging
import obfsproxy.common.transport_config as transport_config

from pyptlib.client import ClientTransportPlugin
from pyptlib.config import EnvError

import pprint

log = logging.get_obfslogger()

def do_managed_client():
    """Start the managed-proxy protocol as a client."""

    should_start_event_loop = False

    ptclient = ClientTransportPlugin()
    try:
        ptclient.init(transports.transports.keys())
    except EnvError, err:
        log.warning("Client managed-proxy protocol failed (%s)." % err)
        return

    log.debug("pyptlib gave us the following data:\n'%s'", pprint.pformat(ptclient.getDebugData()))

    # Apply the proxy settings if any
    proxy = ptclient.config.getProxy()
    if proxy:
        # Make sure that we have all the necessary dependencies
        try:
            network.ensure_outgoing_proxy_dependencies()
        except network.OutgoingProxyDepsFailure, err:
            ptclient.reportProxyError(str(err))
            return

        ptclient.reportProxySuccess()

    for transport in ptclient.getTransports():

        # Will hold configuration parameters for the pluggable transport module.
        pt_config = transport_config.TransportConfig()
        pt_config.setStateLocation(ptclient.config.getStateLocation())
        pt_config.setListenerMode("socks")
        pt_config.setObfsproxyMode("managed")
        pt_config.setProxy(proxy)

        # Call setup() method for this transport.
        transport_class = transports.get_transport_class(transport, 'socks')
        try:
            transport_class.setup(pt_config)
        except base.TransportSetupFailed, err:
            log.warning("Transport '%s' failed during setup()." % transport)
            ptclient.reportMethodError(transport, "setup() failed: %s." % (err))
            continue

        try:
            addrport = launch_transport.launch_transport_listener(transport, None, 'socks', None, pt_config)
        except transports.TransportNotFound:
            log.warning("Could not find transport '%s'" % transport)
            ptclient.reportMethodError(transport, "Could not find transport.")
            continue
        except error.CannotListenError, e:
            error_msg = "Could not set up listener (%s:%s) for '%s' (%s)." % \
                        (e.interface, e.port, transport, e.socketError[1])
            log.warning(error_msg)
            ptclient.reportMethodError(transport, error_msg)
            continue

        should_start_event_loop = True
        log.debug("Successfully launched '%s' at '%s'" % (transport, log.safe_addr_str(str(addrport))))
        ptclient.reportMethodSuccess(transport, "socks5", addrport, None, None)

    ptclient.reportMethodsEnd()

    if should_start_event_loop:
        log.info("Starting up the event loop.")
        reactor.run()
    else:
        log.info("No transports launched. Nothing to do.")
