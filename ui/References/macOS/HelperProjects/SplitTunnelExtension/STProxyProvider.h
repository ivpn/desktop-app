//
//  STProxyProvider.h
//
//  A "transparent proxy" provider: macOS hands us every outbound network
//  flow (subject to the rules we register in -startProxyWithOptions:), and
//  for each one we decide whether to let the OS handle it as usual, or take
//  it over ourselves and relay its bytes over a network interface of our
//  choosing. That per-flow choice is the entire mechanism Split Tunnel is
//  built on.
//
//  This class is instantiated by the OS - see main.m - never by our own code.
//
#import <NetworkExtension/NetworkExtension.h>

@interface STProxyProvider : NETransparentProxyProvider
@end
