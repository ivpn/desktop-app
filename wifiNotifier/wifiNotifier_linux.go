// +build linux

package wifiNotifier

/*
#include <stdio.h>  // printf
#include <string.h> // strdup prototype
#include <stdlib.h> // free protype

#include <netinet/in.h>
#include <linux/netlink.h>
#include <linux/rtnetlink.h>
#include <net/if.h>
#include <arpa/inet.h>

#include <sys/types.h>
#include <sys/socket.h>

#include <iwlib.h> // sudo apt-get install libiw-dev
#include <ifaddrs.h>

static inline char* get_essid (char *iface)
{
   int           fd;
   struct iwreq  w;
   char          essid[IW_ESSID_MAX_SIZE];
   if (!iface) return NULL;

   fd = socket(AF_INET, SOCK_DGRAM, 0);

   strncpy (w.ifr_ifrn.ifrn_name, iface, IFNAMSIZ);
   memset (essid, 0, IW_ESSID_MAX_SIZE);
   w.u.essid.pointer = (caddr_t *) essid;
   w.u.data.length = IW_ESSID_MAX_SIZE;
   w.u.data.flags = 0;

   int isOK = ioctl (fd, SIOCGIWESSID, &w);
   close (fd);

   if (isOK != 0) return NULL;

   return strdup (essid);
}

static inline char * getCurrentSSID(void) {
    char* retSSID = NULL;

    // get all available network interfaces
    struct ifaddrs *addrs,*tmp_addrs;
    getifaddrs(&addrs);
    tmp_addrs = addrs;
    while (tmp_addrs)
    {
        if (tmp_addrs->ifa_addr && tmp_addrs->ifa_addr->sa_family == AF_PACKET)
        {
            retSSID = get_essid (tmp_addrs->ifa_name);

            // do not forget to free 'retSSID' from memory!
            if (retSSID!=NULL) break;
        }

        tmp_addrs = tmp_addrs->ifa_next;
    }
    freeifaddrs(addrs);

    return retSSID;
}

static inline int getCurrentNetworkSecurity() {
    // TODO: implement getCurrentNetworkSecurity functionality
    return 0xFFFFFFFF;
}

static inline char* getAvailableSSIDs(void) {
    // TODO: implement getAvailableSSIDs functionality
    return NULL;
}

static inline void onNetworkStateChanged()
{
	extern void __onWifiChanged(char *);
	__onWifiChanged(getCurrentSSID());
}

static inline void setWifiNotifier(void) {
    struct sockaddr_nl addr;
    int sock, len;
    char buffer[4096];
    struct nlmsghdr *nlh;

    if ((sock = socket(PF_NETLINK, SOCK_RAW, NETLINK_ROUTE)) == -1) {
        perror("couldn't open NETLINK_ROUTE socket");
        return;
    }

    memset(&addr, 0, sizeof(addr));
    addr.nl_family = AF_NETLINK;
    addr.nl_groups = RTMGRP_IPV4_IFADDR;

    if (bind(sock, (struct sockaddr *)&addr, sizeof(addr)) == -1) {
        perror("couldn't bind");
        return;
    }

    nlh = (struct nlmsghdr *)buffer;
    while ((len = recv(sock, nlh, 4096, 0)) > 0) {
        while ((NLMSG_OK(nlh, len)) && (nlh->nlmsg_type != NLMSG_DONE)) {
            switch(nlh->nlmsg_type) {
                case RTM_NEWADDR:
                case RTM_DELADDR:
                    onNetworkStateChanged();
            }
            nlh = NLMSG_NEXT(nlh, len);
        }
    }
}
*/
import "C"
import (
	"strings"
	"unsafe"
)

var internalOnWifiChangedCb func(string)

//export __onWifiChanged
func __onWifiChanged(ssid *C.char) {
	goSsid := C.GoString(ssid)
	C.free(unsafe.Pointer(ssid))

	if internalOnWifiChangedCb != nil {
		internalOnWifiChangedCb(goSsid)
	}
}

// GetAvailableSSIDs returns the list of the names of available Wi-Fi networks
func GetAvailableSSIDs() []string {
	ssidList := C.getAvailableSSIDs()
	goSsidList := C.GoString(ssidList)
	C.free(unsafe.Pointer(ssidList))
	return strings.Split(goSsidList, "\n")
}

// GetCurrentSSID returns current WiFi SSID
func GetCurrentSSID() string {
	ssid := C.getCurrentSSID()
	goSsid := C.GoString(ssid)
	C.free(unsafe.Pointer(ssid))
	return goSsid
}

// GetCurrentNetworkSecurity returns current security mode
func GetCurrentNetworkSecurity() WiFiSecurity {
	return WiFiSecurity(C.getCurrentNetworkSecurity())
}

// SetWifiNotifier initializes a handler method 'OnWifiChanged'
func SetWifiNotifier(cb func(string)) {
	internalOnWifiChangedCb = cb
	go C.setWifiNotifier()
}
