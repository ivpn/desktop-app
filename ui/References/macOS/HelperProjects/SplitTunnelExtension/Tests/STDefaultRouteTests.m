//
//  STDefaultRouteTests.m
//
//  Tests for the routing-table scan behind the default-route interface
//  detection in STPhysicalInterfaceSelector.m, fed hand-built NET_RT_DUMP
//  tables so both address families are covered regardless of the network
//  the tests run on. Run by Tests/run_tests.sh.
//
#import <Foundation/Foundation.h>
#import <net/if.h>
#import <net/route.h>
#import <netinet/in.h>
#import <sys/socket.h>
#import <sys/sysctl.h>
#import "../STPhysicalInterfaceSelector.h"

static int gRouteFailures = 0;

#define EXPECT(cond, ...) do { \
    if (!(cond)) { gRouteFailures++; NSLog(@"FAIL line %d: %@", __LINE__, [NSString stringWithFormat:__VA_ARGS__]); } \
    else { NSLog(@"ok   %s", #cond); } \
} while (0)

// One routing message: destination, gateway and netmask of `family`, on the
// interface `ifname`. `maskFirstByte` is the first byte of the network mask
// (0 for a /0 'default', anything else for a narrower route).
static NSData *RouteMessage(int family, const char *ifname, int flags, unsigned char maskFirstByte) {
    size_t saLen = (family == AF_INET6) ? sizeof(struct sockaddr_in6) : sizeof(struct sockaddr_in);
    NSMutableData *msg = [NSMutableData dataWithLength:sizeof(struct rt_msghdr) + 3 * saLen];
    struct rt_msghdr *rtm = msg.mutableBytes;
    rtm->rtm_msglen = (u_short)msg.length;
    rtm->rtm_version = RTM_VERSION;
    rtm->rtm_type = RTM_GET;
    rtm->rtm_index = (u_short)if_nametoindex(ifname);
    rtm->rtm_flags = flags;
    rtm->rtm_addrs = RTA_DST | RTA_GATEWAY | RTA_NETMASK;
    unsigned char *sa = (unsigned char *)(rtm + 1);
    for (int i = 0; i < 3; i++, sa += saLen) {
        ((struct sockaddr *)sa)->sa_len = (uint8_t)saLen;
        ((struct sockaddr *)sa)->sa_family = (sa_family_t)family;
    }
    // Netmask is the third sockaddr; the address bytes start at the family's offset.
    unsigned char *mask = (unsigned char *)(rtm + 1) + 2 * saLen;
    size_t addrOffset = (family == AF_INET6) ? offsetof(struct sockaddr_in6, sin6_addr) : offsetof(struct sockaddr_in, sin_addr);
    mask[addrOffset] = maskFirstByte;
    return msg;
}

static NSString *Scan(NSArray<NSData *> *messages, int family) {
    NSMutableData *table = [NSMutableData data];
    for (NSData *m in messages) { [table appendData:m]; }
    return STDefaultRouteInterfaceNameInTable(table.bytes, table.length, family);
}

static void LogLiveTable(int family) {
    int mib[6] = { CTL_NET, PF_ROUTE, 0, family, NET_RT_DUMP, 0 };
    size_t needed = 0;
    if (sysctl(mib, 6, NULL, &needed, NULL, 0) < 0) { return; }
    char *buf = malloc(needed);
    if (buf && sysctl(mib, 6, buf, &needed, NULL, 0) == 0) {
        // Depends on the machine's network, so only reported, never asserted.
        NSLog(@"info live %s default-route interface: %@", family == AF_INET6 ? "IPv6" : "IPv4",
              STDefaultRouteInterfaceNameInTable(buf, needed, family) ?: @"(none)");
    }
    free(buf);
}

int STDefaultRouteTestsRun(void) {
    // en0 is Ethernet-class on every Mac, lo0 is not.
    if (if_nametoindex("en0") == 0) { NSLog(@"skip: no en0 on this machine"); return 0; }

    for (int family = AF_INET; ; family = AF_INET6) {
        NSData *unscopedEn0 = RouteMessage(family, "en0", RTF_UP | RTF_GATEWAY, 0);
        NSData *scopedEn0   = RouteMessage(family, "en0", RTF_UP | RTF_GATEWAY | RTF_IFSCOPE, 0);
        NSData *tunnel      = RouteMessage(family, "lo0", RTF_UP | RTF_GATEWAY, 0);
        NSData *halfDefault = RouteMessage(family, "en0", RTF_UP | RTF_GATEWAY, 0x80); // the VPN's "0/1" capture route
        NSData *down        = RouteMessage(family, "en0", RTF_GATEWAY, 0);

        EXPECT([Scan(@[unscopedEn0], family) isEqualToString:@"en0"], @"family %d: unscoped default on en0", family);
        EXPECT([Scan(@[scopedEn0], family) isEqualToString:@"en0"], @"family %d: scoped default is accepted as fallback", family);
        EXPECT([Scan(@[tunnel, scopedEn0], family) isEqualToString:@"en0"], @"family %d: non-physical default is skipped", family);
        EXPECT(Scan(@[tunnel], family) == nil, @"family %d: only a non-physical default", family);
        EXPECT(Scan(@[halfDefault], family) == nil, @"family %d: /1 route is not a default", family);
        EXPECT(Scan(@[down], family) == nil, @"family %d: route that is not up", family);
        EXPECT(Scan(@[], family) == nil, @"family %d: empty table", family);
        EXPECT(Scan(@[unscopedEn0], family == AF_INET ? AF_INET6 : AF_INET) == nil, @"family %d: other family's routes are ignored", family);
        if (family == AF_INET6) { break; }
    }

    LogLiveTable(AF_INET);
    LogLiveTable(AF_INET6);
    return gRouteFailures;
}
