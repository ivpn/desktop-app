import obfsproxy.network.network as network
import obfsproxy.transports.transports as transports
import obfsproxy.network.socks as socks
import obfsproxy.network.extended_orport as extended_orport

from twisted.internet import reactor

def launch_transport_listener(transport, bindaddr, role, remote_addrport, pt_config, ext_or_cookie_file=None):
    """
    Launch a listener for 'transport' in role 'role' (socks/client/server/ext_server).

    If 'bindaddr' is set, then listen on bindaddr. Otherwise, listen
    on an ephemeral port on localhost.
    'remote_addrport' is the TCP/IP address of the other end of the
    circuit. It's not used if we are in 'socks' role.

    'pt_config' contains configuration options (such as the state location)
    which are of interest to the pluggable transport.

    'ext_or_cookie_file' is the filesystem path where the Extended
    ORPort Authentication cookie is stored. It's only used in
    'ext_server' mode.

    Return a tuple (addr, port) representing where we managed to bind.

    Throws obfsproxy.transports.transports.TransportNotFound if the
    transport could not be found.

    Throws twisted.internet.error.CannotListenError if the listener
    could not be set up.
    """

    transport_class = transports.get_transport_class(transport, role)
    listen_host = bindaddr[0] if bindaddr else 'localhost'
    listen_port = int(bindaddr[1]) if bindaddr else 0

    if role == 'socks':
        factory = socks.OBFSSOCKSv5Factory(transport_class, pt_config)
    elif role == 'ext_server':
        assert(remote_addrport and ext_or_cookie_file)
        factory = extended_orport.ExtORPortServerFactory(remote_addrport, ext_or_cookie_file, transport, transport_class, pt_config)
    else:
        assert(remote_addrport)
        factory = network.StaticDestinationServerFactory(remote_addrport, role, transport_class, pt_config)

    addrport = reactor.listenTCP(listen_port, factory, interface=listen_host)

    return (addrport.getHost().host, addrport.getHost().port)
