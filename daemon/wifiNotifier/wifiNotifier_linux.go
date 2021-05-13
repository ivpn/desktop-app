// +build linux

package wifiNotifier

/*
// Trying to avoid using dynamic linking, therefore disabled 'iwlib'
// (wireless-tools library, which is requires to have installed correspond package).
// If you want to use original iwlib package (and do not use custom 'linux_iwlib_2.c'):
// 1) uncomment '#cgo LDFLAGS: -liw'
// 2) comment '#include "iwlib_2_linux.c"'
// 3) remove  suffix '_2' from function names (in this file): iw_get_range_info_2, iw_init_event_stream_2, iw_extract_event_stream_2
// #cgo LDFLAGS: -liw
#include "iwlib_2_linux.c"

#include <stdio.h>  // printf
#include <string.h> // strndup prototype
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

static inline char* concatenate(char* baseString, const char* toAdd, char delimiter) {
	if (toAdd == NULL)
		return baseString;
	int addingLen = strlen(toAdd);
	if (addingLen == 0)
		return baseString;

	if (baseString == NULL) {
		baseString = (char*)malloc(addingLen +1);

		memset(baseString, 0, addingLen + 1);
		strcpy(baseString, toAdd);
		return baseString;
	}

	int newSize = strlen(baseString) + ((delimiter != 0) ? 1 : 0) + addingLen + 1;
	char* newString = (char*)malloc(newSize);

	if (delimiter != 0)
		sprintf(newString, "%s%c%s", baseString, delimiter, toAdd);
	else
		sprintf(newString, "%s%s", baseString, toAdd);

	free(baseString);

	return newString;
}

static inline char*  scanSSIDList(const char* interfaceName, int *retIsInsecure, const char* ssidToCheckSecurity) {
    char *ret = NULL;

    int sockfd = socket(AF_INET, SOCK_DGRAM, 0);
    if (sockfd == -1)
        return NULL;

    //---------------------------------------------------------------------

    struct iw_range range;

    if ((iw_get_range_info_2(sockfd, interfaceName, &range) < 0) ||
        (range.we_version_compiled < 14))
    {
        close(sockfd);
        return NULL; // interface doesn't support scanning
    }

    __u8 wev = range.we_version_compiled;

    //---------------------------------------------------------------------

    struct iwreq request;
    memset(&request, 0, sizeof(request));
    request.u.param.flags = IW_SCAN_DEFAULT;
    request.u.param.value = 0;

    if (iw_set_ext(sockfd, interfaceName, SIOCSIWSCAN, &request) == -1)
    {
        close(sockfd);
        return NULL;
    }

    //---------------------------------------------------------------------

    struct timeval startTime, endTime, diffTime = { 0, 0 };
    gettimeofday(&startTime, NULL);

    char scanBuffer[0xFFFF];

    int replyFound = 0;
    while (replyFound == 0)
    {
        memset(scanBuffer, 0, sizeof(scanBuffer));
        request.u.data.pointer = scanBuffer;
        request.u.data.length = sizeof(scanBuffer);
        request.u.data.flags = 0;

        int result = iw_get_ext(sockfd,
                                interfaceName,
                                SIOCGIWSCAN,
                                &request);

        if (result == -1 && errno != EAGAIN)
        {
            close(sockfd);
            return NULL;
        }

        if (result == 0)
        {
            replyFound = 1;
            break;
        }

        gettimeofday(&endTime, NULL);
        timersub(&endTime, &startTime, &diffTime);
        if (diffTime.tv_sec > 10)
            break;

        usleep(100000);
    }
    close(sockfd);

    //---------------------------------------------------------------------

    if (replyFound)
    {
        struct iw_event iwe;
        struct stream_descr stream;

        iw_init_event_stream_2(&stream,
                             scanBuffer,
                             request.u.data.length);

        char eventBuffer[512] = {0};

        char essid[IW_ESSID_MAX_SIZE+1];
        unsigned short encodeFlags = -1;
        while (iw_extract_event_stream_2(&stream, &iwe, wev) > 0)
        {
            switch (iwe.cmd)
            {
                case SIOCGIWESSID:
                {
                    memset(essid, 0, sizeof(essid));
                    if((iwe.u.essid.pointer) && (iwe.u.essid.length))
                    {
                        memcpy(essid,
                            iwe.u.essid.pointer,
                            iwe.u.essid.length);

                        essid[iwe.u.essid.length] = 0;
                        ret = concatenate(ret, essid, '\n');

                        if (retIsInsecure!=NULL
                            && ssidToCheckSecurity!=NULL
                            && encodeFlags != -1
                            && strcmp(essid, ssidToCheckSecurity)==0)
                        {
                            *retIsInsecure = ( encodeFlags & IW_ENCODE_DISABLED ) > 0; // IW_ENCODE_OPEN
                            encodeFlags = -1;
                        }
                    }
                }
                break;

                case SIOCGIWENCODE:
                {
                    encodeFlags = iwe.u.encoding.flags;
                    break;
                }
            }
        }
    }

    return ret;
}

static inline char* get_essid (char *iface)
{
   int           fd;
   struct iwreq  w;
   char          essid[IW_ESSID_MAX_SIZE+1];
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

   return strndup (essid, 32); // normally, the IW_ESSID_MAX_SIZE is 32 bytes (the coping with potential security flaws within the driver)
}

static inline char * getCurrentWifiInfo(int* retIsInsecure) {
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
            if (retSSID!=NULL)
            {
                if (retIsInsecure!=NULL) {
                    char* wifiList = scanSSIDList(tmp_addrs->ifa_name, retIsInsecure, retSSID);
                    if (wifiList!=NULL) free(wifiList);
                }
                break;
            }
        }

        tmp_addrs = tmp_addrs->ifa_next;
    }
    freeifaddrs(addrs);

    return retSSID;
}

static inline char * getCurrentSSID(void) {
    return getCurrentWifiInfo(NULL);
}

static inline int getCurrentNetworkIsInsecure() {
    int retIsInecure = 0xFFFFFFFF;
    char* ssid = getCurrentWifiInfo(&retIsInecure);
    if (ssid!=NULL) free(ssid);
    return retIsInecure;
}

static inline char* getAvailableSSIDs(void) {
    char* retSSID = NULL;

    // get all available network interfaces
    struct ifaddrs *addrs,*tmp_addrs;
    getifaddrs(&addrs);
    tmp_addrs = addrs;
    while (tmp_addrs)
    {
        if (tmp_addrs->ifa_addr && tmp_addrs->ifa_addr->sa_family == AF_PACKET)
            retSSID = concatenate(retSSID, scanSSIDList(tmp_addrs->ifa_name, NULL, NULL), '\n');
        tmp_addrs = tmp_addrs->ifa_next;
    }
    freeifaddrs(addrs);

    return retSSID;
}
*/
import "C"
import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/ivpn/desktop-app-daemon/oshelpers/linux/netlink"
)

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

// GetCurrentNetworkIsInsecure returns current security mode
func GetCurrentNetworkIsInsecure() bool {
	// TODO: implement GetCurrentNetworkIsInsecure functionality for Linux
	return false

	// ret := WiFiSecurityUnknown
	// if C.getCurrentNetworkIsInsecure() == 1 {
	// 	ret = WiFiSecurityNone
	// }
	// return ret
}

// SetWifiNotifier initializes a handler method 'OnWifiChanged'
func SetWifiNotifier(cb func(string)) error {
	if cb == nil {
		return fmt.Errorf("callback function not defined")
	}

	l, err := netlink.CreateListener()
	if err != nil {
		return fmt.Errorf("Netlink listener initialization error: %w", err)
	}
	go func() {
		for {
			msgs, err := l.ReadMsgs()
			if err != nil {
				fmt.Println("Could not read netlink messages: %s", err)
			}
			for _, m := range msgs {
				if netlink.IsNewAddr(&m) || netlink.IsDelAddr(&m) {
					cb(GetCurrentSSID())
				}
			}
		}
	}()
	return nil
}
