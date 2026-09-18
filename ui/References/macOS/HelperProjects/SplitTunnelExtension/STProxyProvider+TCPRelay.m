//
//  STProxyProvider+TCPRelay.m
//
//  For TCP, one app-level flow = one relay connection (1:1). Opens a second,
//  independent TCP connection to the flow's original destination - but
//  pinned to the physical network interface - and copies bytes in both
//  directions between the app's flow and that connection.
//
#import "STProxyProvider+Private.h"
#import "STPhysicalInterfaceSelector.h"
#import "STLog.h"

@implementation STProxyProvider (TCPRelay)

- (void)relayTCPFlow:(NEAppProxyTCPFlow *)flow {
    // flow.remoteEndpoint is a Foundation-style NWHostEndpoint. The
    // interface-pinning API we need below (nw_parameters_*) is a different,
    // lower-level C API with its own endpoint type (nw_endpoint_t) - this is
    // just bridging the destination address/port between the two.
    NWHostEndpoint *remote = (NWHostEndpoint *)flow.remoteEndpoint;
    if (![remote isKindOfClass:[NWHostEndpoint class]] || remote.hostname.length == 0 || remote.port.length == 0) {
        STLogError(@"Unexpected/incomplete remote endpoint (%@ %@:%@), dropping flow", NSStringFromClass([flow.remoteEndpoint class]), remote.hostname, remote.port);
        [flow closeReadWithError:nil];
        [flow closeWriteWithError:nil];
        return;
    }
    STLogDebug(@"TCP relay target: %@:%@", remote.hostname, remote.port);

    if (![self.interfaceSelector isCurrentlySatisfied]) {
        // The selected physical interface/type is currently unavailable
        // (e.g. Wi-Fi manually turned off) - refuse the flow immediately
        // instead of dialing against a requirement that can't succeed right
        // now. Self-heals: the next new flow after the interface/type comes
        // back is relayed normally, no restart needed.
        STLogInfo(@"Physical interface/type currently unavailable, refusing TCP flow to %@:%@", remote.hostname, remote.port);
        [flow closeReadWithError:nil];
        [flow closeWriteWithError:nil];
        return;
    }

    nw_endpoint_t endpoint = nw_endpoint_create_host(remote.hostname.UTF8String, remote.port.UTF8String);

    nw_parameters_t parameters = nw_parameters_create_secure_tcp(
        NW_PARAMETERS_DISABLE_PROTOCOL,      // no TLS at this layer - we relay raw bytes; the app does its own TLS if needed
        NW_PARAMETERS_DEFAULT_CONFIGURATION); // default TCP options

    // Forces this outgoing connection onto the physical network (pinned
    // interface, requested type, or auto-detected type - see
    // STPhysicalInterfaceSelector) instead of letting it follow the
    // system's current default route (which, with a VPN connected, would be
    // the VPN's virtual "utun" interface). This is the mechanism the whole
    // feature relies on.
    [self.interfaceSelector applyRequirementToParameters:parameters];

    nw_connection_t connection = nw_connection_create(endpoint, parameters);
    // One dedicated queue per TCP flow, instead of one shared across the
    // whole process - lets unrelated flows relay concurrently. Label is only
    // ever seen in the debugger/Instruments/crash logs, nothing external
    // matches on it, so a plain string is fine here.
    dispatch_queue_t connectionQueue = dispatch_queue_create("relay.tcp", DISPATCH_QUEUE_SERIAL);
    nw_connection_set_queue(connection, connectionQueue);

    [self st_registerTCPFlow:flow connection:connection];

    __weak typeof(self) weakSelf = self;
    nw_connection_set_state_changed_handler(connection, ^(nw_connection_state_t state, nw_error_t error) {
        __strong typeof(self) strongSelf = weakSelf;
        if (!strongSelf) { return; }

        if (state == nw_connection_state_ready) {
            // Only after the real connection is up do we tell NetworkExtension
            // we're accepting the flow; from this point on the app can
            // actually send/receive data through us.
            [flow openWithLocalEndpoint:nil completionHandler:^(NSError * _Nullable openError) {
                if (openError) {
                    STLogError(@"Failed to open TCP flow after connecting: %@", openError);
                    nw_connection_cancel(connection);
                    return;
                }
                STLogDebug(@"TCP relay established");
                // The two directions finish independently and
                // asynchronously - this group defers the actual cancel +
                // full teardown (-teardownTCPFlow:connection:) until BOTH
                // have finished, instead of one direction unilaterally
                // closing a connection the other side might still be using.
                // Only created here (not earlier) so a connection that never
                // reaches "ready" never has an unbalanced group left behind -
                // that case is already handled directly, below.
                dispatch_group_t doneGroup = dispatch_group_create();
                dispatch_group_enter(doneGroup); // left by pumpFlow:toConnection:doneGroup: (app -> network)
                dispatch_group_enter(doneGroup); // left by pumpConnection:toFlow:doneGroup: (network -> app)
                dispatch_group_notify(doneGroup, connectionQueue, ^{
                    [weakSelf teardownTCPFlow:flow connection:connection];
                });
                [strongSelf pumpFlow:flow toConnection:connection doneGroup:doneGroup];
                [strongSelf pumpConnection:connection toFlow:flow doneGroup:doneGroup];
            }];
        } else if (state == nw_connection_state_failed || state == nw_connection_state_cancelled) {
            STLogDebug(@"TCP relay connection ended (state=%ld, error=%@)", (long)state, error);
            [strongSelf teardownTCPFlow:flow connection:connection];
        }
    });

    nw_connection_start(connection);
}

// Reads a chunk from the app (the "flow") and writes it to the real network
// (the "connection"), then schedules itself again for the next chunk. There
// is no blocking loop here - each completion handler kicks off the next
// read, which is the normal pattern for these APIs. Leaves `doneGroup`
// exactly once, on whichever terminal branch this direction ends on.
- (void)pumpFlow:(NEAppProxyTCPFlow *)flow toConnection:(nw_connection_t)connection doneGroup:(dispatch_group_t)doneGroup {
    __weak typeof(self) weakSelf = self;
    [flow readDataWithCompletionHandler:^(NSData * _Nullable data, NSError * _Nullable error) {
        @autoreleasepool {
        __strong typeof(self) strongSelf = weakSelf;
        if (!strongSelf) { return; }

        if (error || data.length == 0) {
            // EOF (or error) from the app side: tell the real connection
            // there is no more data coming from us. This direction only
            // counts as done once that final message has actually been
            // sent (or failed to send) - either way, nothing more to do.
            nw_connection_send(connection, NULL, NW_CONNECTION_FINAL_MESSAGE_CONTEXT, true, ^(nw_error_t sendError) {
                dispatch_group_leave(doneGroup);
            });
            return;
        }

        // Custom destructor just keeps `data` alive until the send
        // completes, instead of DISPATCH_DATA_DESTRUCTOR_DEFAULT's
        // unconditional memcpy of every chunk relayed.
        dispatch_data_t payload = dispatch_data_create(data.bytes, data.length,
                                                        dispatch_get_global_queue(DISPATCH_QUEUE_PRIORITY_DEFAULT, 0),
                                                        ^{ (void)data; });
        nw_connection_send(connection, payload, NW_CONNECTION_DEFAULT_MESSAGE_CONTEXT, true, ^(nw_error_t sendError) {
            if (sendError) {
                STLogError(@"Send to physical interface failed: %@", sendError);
                dispatch_group_leave(doneGroup); // this direction stops here, same as a clean EOF
                return;
            }
            [strongSelf pumpFlow:flow toConnection:connection doneGroup:doneGroup]; // keep reading
        });
        }
    }];
}

// The mirror image of the method above: reads from the real network and
// writes back to the app. Leaves `doneGroup` exactly once, on whichever
// terminal branch this direction ends on.
- (void)pumpConnection:(nw_connection_t)connection toFlow:(NEAppProxyTCPFlow *)flow doneGroup:(dispatch_group_t)doneGroup {
    // Stream data has no natural message boundary, so this is just a
    // deliberate cap on worst-case memory/latency per completion - not a
    // protocol requirement, unlike UDP's datagram-size bound below.
    static const uint32_t kMaxReceiveLength = 64 * 1024;
    __weak typeof(self) weakSelf = self;
    nw_connection_receive(connection, 1, kMaxReceiveLength,
        ^(dispatch_data_t _Nullable content, nw_content_context_t _Nullable context, bool isComplete, nw_error_t _Nullable receiveError) {
        @autoreleasepool {
        __strong typeof(self) strongSelf = weakSelf;
        if (!strongSelf) { return; }

        void (^finishOrContinue)(void) = ^{
            if (isComplete || receiveError) {
                // Half-close only: closes the flow's write side (network's
                // side of the conversation is done), but does NOT cancel
                // `connection` here - pumpFlow:toConnection:doneGroup: (the
                // app -> network direction) may still be uploading. The
                // dispatch_group is what actually triggers the real cancel +
                // full teardown, once BOTH directions have finished.
                [flow closeWriteWithError:nil];
                dispatch_group_leave(doneGroup);
            } else {
                [strongSelf pumpConnection:connection toFlow:flow doneGroup:doneGroup]; // keep receiving
            }
        };

        if (content) {
            // dispatch_data_t and NSData are documented by Apple as
            // "toll-free bridged" on Apple platforms - this cast just
            // relabels the same bytes, no copy happens here.
            NSData *data = (NSData *)content;
            [flow writeData:data withCompletionHandler:^(NSError * _Nullable writeError) {
                if (writeError) {
                    STLogError(@"Failed to write TCP data back to the app: %@", writeError);
                }
                finishOrContinue();
            }];
        } else {
            finishOrContinue();
        }
        }
    });
}

// Cancels the physical-interface connection, closes the app-facing flow
// (both directions), and drops both from the shared bookkeeping. Used when
// the connection itself has ended, not for the network-side half-close
// handled in pumpConnection:toFlow:'s finishOrContinue above.
- (void)teardownTCPFlow:(NEAppProxyTCPFlow *)flow connection:(nw_connection_t)connection {
    if (connection) { nw_connection_cancel(connection); }
    [flow closeReadWithError:nil];
    [flow closeWriteWithError:nil];
    [self st_unregisterTCPFlow:flow];
}

@end
