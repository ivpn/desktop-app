
#ifndef __LIBIVPN_H__
#define __LIBIVPN_H__

#include <xpc/xpc.h>
#include <ServiceManagement/ServiceManagement.h>
#include <SystemConfiguration/SystemConfiguration.h>
#include <Security/Security.h>

#include <sys/sysctl.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <net/if.h>
#include <net/if_var.h>
#include <net/if_dl.h>
#include <net/if_types.h>
#include <net/if_mib.h>
#include <net/route.h>

#include <netinet/in.h>
#include <netinet/in_var.h>

#define LIBIVPN_XPC_MESSAGE_TYPE_START_REQUEST 1
#define LIBIVPN_XPC_MESSAGE_TYPE_STARTED_REPLY 2

#define HELPER_LATEST_VERSION 0
#define HELPER_NOT_INSTALLED 1
#define HELPER_UPGRADE_REQUIRED 2

typedef void (*AgentConnectedHandler) (int port, uint64_t secret);

void start_xpc_listener(char *name, int serviceTcpPort, uint64_t serviceSecret);
void connect_to_agent(char *name, AgentConnectedHandler handler);
void close_connection();

int install_helper(char *label);
int install_helper_with_auth(char *label, AuthorizationRef authRef);

int remove_helper(char *label);
int remove_helper_with_auth(char *label, AuthorizationRef authRef);

int register_network_change_monitor();
void remove_network_change_monitor();

int wait_for_route_change();

int get_interface_statistics(
  char *interface, u_int32_t *bytesReceived, u_int32_t *bytesSent);

#endif
