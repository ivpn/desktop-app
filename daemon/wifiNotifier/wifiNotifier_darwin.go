//go:build darwin && !nowifi
// +build darwin,!nowifi

package wifiNotifier

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework SystemConfiguration -framework CoreWLAN -framework Foundation

#include <stdlib.h>
#import <Foundation/Foundation.h>
#import <CoreWLAN/CoreWLAN.h>
#import <SystemConfiguration/SystemConfiguration.h>

static inline  char* nsstring2cstring(NSString *s){
    if (s == NULL) { return NULL; }

	char *cstr = strdup([s UTF8String]);
    return cstr;
}

#define NOT_CONNECTED @""

static inline CWInterface * getCWInterface() {
	CWWiFiClient *swc = [CWWiFiClient sharedWiFiClient];
	if (swc == nil) return nil;
	return [swc interface];
}

static inline void wifi_network_changed(SCDynamicStoreRef store, CFArrayRef changedKeys, void *ctx)
{
	extern void __onWifiChanged();
	__onWifiChanged();
}

static inline char * getCurrentSSID(void) {
	CWInterface * WiFiInterface = getCWInterface();
	if (WiFiInterface == nil) return nsstring2cstring(NOT_CONNECTED);

	NSString *ssid = [WiFiInterface ssid] ? [WiFiInterface ssid] : NOT_CONNECTED;
	return nsstring2cstring(ssid);
}

static inline int getCurrentNetworkSecurity() {
	CWInterface * WiFiInterface = getCWInterface();
	if (WiFiInterface == nil) return 0xFFFFFFFF;

	return [WiFiInterface security];
}

static inline char* getAvailableSSIDs(void) {
	CWInterface * WiFiInterface = getCWInterface();
	if (WiFiInterface == nil) return nil;

	NSError *err = nil;
	NSSet *scanset = [WiFiInterface scanForNetworksWithSSID:Nil error:&err];
    if (err!=nil || scanset == nil || scanset.count == 0) return nil;

	NSString *retString = nil;
	int i=0;
	for (CWNetwork * nw in scanset)
    {
		if (nw == nil || [nw ssid] == nil) continue;
		NSString * ssid = [[[nw ssid] componentsSeparatedByCharactersInSet:[NSCharacterSet newlineCharacterSet]] componentsJoinedByString:@" "];
		if (i++ == 0)
			retString = ssid;
		else
			retString = [NSString stringWithFormat:@"%@\n%@", retString , ssid];
	}

	return nsstring2cstring(retString);
}

static inline void setWifiNotifier(void) {
    CWInterface * WiFiInterface = getCWInterface();
	if (WiFiInterface == nil) return;

	NSArray* arr = [CWWiFiClient interfaceNames];
	NSSet *wifiInterfaces = [NSSet setWithArray:arr];

    NSMutableArray *scKeys = [[NSMutableArray alloc] init];
    [wifiInterfaces enumerateObjectsUsingBlock:^(NSString *ifName, BOOL *stop)
     {
         [scKeys addObject: [NSString stringWithFormat:@"State:/Network/Interface/%@/AirPort", ifName]];
     }];

    SCDynamicStoreContext ctx = { 0, NULL, NULL, NULL, NULL };
    SCDynamicStoreRef store = SCDynamicStoreCreate(kCFAllocatorDefault, CFSTR("IVPN"), wifi_network_changed, &ctx);

    SCDynamicStoreSetNotificationKeys(store, (__bridge CFArrayRef)scKeys, NULL);

    CFRunLoopSourceRef src = SCDynamicStoreCreateRunLoopSource(kCFAllocatorDefault, store, 0);
	CFRunLoopAddSource([[NSRunLoop currentRunLoop] getCFRunLoop], src, kCFRunLoopCommonModes);

	CFRunLoopRun();
}
*/
import "C"
import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

var internalOnWifiChangedCb func()

func init() {
	ex_initIsNativeApiWorks()
}

//export __onWifiChanged
func __onWifiChanged() {
	if internalOnWifiChangedCb != nil {
		internalOnWifiChangedCb()
	}
}

// GetAvailableSSIDs returns the list of the names of available Wi-Fi networks
func implGetAvailableSSIDs() []string {
	if !ex_nativeApiWorks {
		return ex_getAvailableSSIDs()
	}

	ssidList := C.getAvailableSSIDs()
	goSsidList := C.GoString(ssidList)
	C.free(unsafe.Pointer(ssidList))
	return strings.Split(goSsidList, "\n")
}

// GetCurrentWifiInfo returns current WiFi info
func implGetCurrentWifiInfo() (WifiInfo, error) {
	SSID := getCurrentSSID()

	// If we can not obtain SSID using native API - we use external tool as a workaround
	if !ex_nativeApiWorks {
		if len(SSID) > 0 {
			log.Info("Native API works!")
			ex_nativeApiWorks = true // Looks like native API works
		} else {
			return ex_getWifiInfo() // We can not use native API for SSID detection, so we use external tool as a workaround
		}
	}

	return WifiInfo{
		SSID:       SSID,
		IsInsecure: getCurrentNetworkIsInsecure(),
	}, nil
}

// GetCurrentSSID returns current WiFi SSID
func getCurrentSSID() string {
	ssid := C.getCurrentSSID()
	goSsid := C.GoString(ssid)
	C.free(unsafe.Pointer(ssid))
	return goSsid
}

// GetCurrentNetworkIsInsecure returns current security mode
func getCurrentNetworkIsInsecure() bool {
	const (
		CWSecurityNone               = 0
		CWSecurityWEP                = 1
		CWSecurityWPAPersonal        = 2
		CWSecurityWPAPersonalMixed   = 3
		CWSecurityWPA2Personal       = 4
		CWSecurityPersonal           = 5
		CWSecurityDynamicWEP         = 6
		CWSecurityWPAEnterprise      = 7
		CWSecurityWPAEnterpriseMixed = 8
		CWSecurityWPA2Enterprise     = 9
		CWSecurityEnterprise         = 10
		CWSecurityWPA3Personal       = 11
		CWSecurityWPA3Enterprise     = 12
		CWSecurityWPA3Transition     = 13
	)

	security := C.getCurrentNetworkSecurity()
	switch security {
	case CWSecurityNone,
		CWSecurityWEP,
		CWSecurityPersonal,
		CWSecurityDynamicWEP:
		return true
	}
	return false
}

// SetWifiNotifier initializes a handler method 'OnWifiChanged'
func implSetWifiNotifier(cb func()) error {
	internalOnWifiChangedCb = cb

	go func() {
		log.Info("WiFi notifier enter")
		defer log.Error("WiFi notifier exit")

		for {
			// Detection WiFi status change in infinite loop.
			// C.setWifiNotifier() should never return.
			//
			// BUT! It can return in some corner cases (e.g. we call it on system boot when WiFi interface still not initialized)
			// In this case - we waiting some delay and trying to call this function again
			C.setWifiNotifier()
			log.Info("Unexpected WiFi notifier exit")
			time.Sleep(time.Second)
			log.Info("WiFi notifier enter. Retry...")
		}
	}()
	return nil
}

// ----------------------------------------------------
// Hacky implementation of obtaining SSID for macOS 14.0+ (Sonoma+)
// ----------------------------------------------------
// Starting from macOS 14 Sonoma release, Apple has changed behavior of CWInterface (CoreWLAN framework):
// It is not possible anymore to obtaine WiFi SSID for background daemons.
// Bellow implementation is a hacky workaround for this issue.
//
// https://developer.apple.com/forums/thread/732431
// https://developer.apple.com/forums/thread/739712#768907022

var ex_nativeApiWorks = true

const ex_airport_tool_bin = "/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport"

func ex_initIsNativeApiWorks() {
	_, err := os.Stat(ex_airport_tool_bin)
	if err != nil {
		log.Debug("!!! airport tool not found !!!")
		return // we can not use airport tool for SSID detection
	}

	// Checking macOS version
	var uts unix.Utsname
	if err := unix.Uname(&uts); err != nil {
		log.Error(fmt.Errorf("Can not obtain macOS version: %w", err))
		return
	}
	release := unix.ByteSliceToString(uts.Release[:])
	dotPos := strings.Index(release, ".")
	if dotPos == -1 {
		log.Error("Can not obtain macOS version")
		return
	}
	major := release[:dotPos]
	majorVersion, err := strconv.Atoi(major)
	if err != nil {
		log.Error(fmt.Errorf("Can not obtain macOS version: %w", err))
		return
	}
	if majorVersion >= 23 {
		// Darwin v23.x.x == macOS 14 Sonoma
		// It is not possible anymore to obtaine WiFi SSID for background daemons since macOS 14.
		ex_nativeApiWorks = false
		log.Warning("macOS 14+ detected. WiFi SSID detection will be performed using external tool")
	}
}

func ex_getWifiInfo() (WifiInfo, error) {
	//log.Debug("!!! Trying to obtain WiFi info using airport tool !!!")

	SSID := ""
	isInsecure := false

	cmd := exec.Command(ex_airport_tool_bin, "--getinfo")
	output, err := cmd.Output()
	if err != nil {
		return WifiInfo{}, err
	}

	const (
		field_ssid = "SSID:"
		field_auth = "link auth:"
	)

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		l := strings.TrimSpace(line)

		if strings.HasPrefix(l, field_ssid) {
			SSID = strings.TrimSpace(strings.TrimPrefix(l, field_ssid))
		}
		if strings.HasPrefix(l, field_auth) {
			auth := strings.TrimSpace(strings.TrimPrefix(l, field_auth))
			if auth == "none" || strings.Contains(auth, "wep") {
				isInsecure = true
			}
		}
	}

	return WifiInfo{SSID: SSID, IsInsecure: isInsecure}, nil
}

func ex_getAvailableSSIDs() []string {
	ret := []string{}

	cmd := exec.Command(ex_airport_tool_bin, "--scan")
	output, err := cmd.Output()
	if err != nil {
		return ret
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if len(line) < 32 {
			break
		}
		if i == 0 {
			// the header has to have first fiels as "SSID". It's length must be 32 symbols
			if strings.TrimSpace(line[:32]) != "SSID" {
				return ret
			}
			continue // skip header
		}
		ssid := strings.TrimSpace(line[:32])
		if ssid != "" {
			ret = append(ret, ssid)
		}
	}
	return ret
}
