package dns

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/ivpn/desktop-app-daemon/netinfo"
	"github.com/ivpn/desktop-app-daemon/service/platform"
)

var (
	_dll = syscall.NewLazyDLL(platform.WindowsNativeHelpersDllPath())
	//_fSetDNSByIndex   = _dll.NewProc("SetDNSByIndex")   // DWORD _cdecl SetDNSByIndex(const WORD interfaceIdx, const char* dnsIP, byte operation)
	_fSetDNSByMAC     = _dll.NewProc("SetDNSByMAC")     // DWORD _cdecl SetDNSByMAC(const char* interfaceMAC, const char* dnsIP, byte operation)
	_fSetDNSByLocalIP = _dll.NewProc("SetDNSByLocalIP") // DWORD _cdecl SetDNSByLocalIP(const char* interfaceLocalAddr, const char* dnsIP, byte operation)
)

var dnsMutex sync.Mutex

// Operation enumerates possible DNS operations
type Operation uint32

// DNS operations
const (
	OperationSet Operation = 0
	OperationAdd Operation = 1
	OperationDel Operation = 2
)

func fSetDNSByMAC(interfaceMACAddr net.HardwareAddr, dns net.IP, op Operation) error {
	dnsString := dns.String()
	if dns.Equal(net.IPv4zero) {
		dnsString = ""
	}

	dnsMutex.Lock()
	defer dnsMutex.Unlock()

	retval, _, err := _fSetDNSByMAC.Call(
		uintptr(unsafe.Pointer(syscall.StringBytePtr(interfaceMACAddr.String()))),
		uintptr(unsafe.Pointer(syscall.StringBytePtr(dnsString))),
		uintptr(op))

	return checkDefaultAPIResp(retval, err)
}

func fSetDNSByLocalIP(interfaceLocalAddr net.IP, dns net.IP, op Operation) error {
	dnsString := dns.String()
	if dns.Equal(net.IPv4zero) {
		dnsString = ""
	}

	dnsMutex.Lock()
	defer dnsMutex.Unlock()

	retval, _, err := _fSetDNSByLocalIP.Call(
		uintptr(unsafe.Pointer(syscall.StringBytePtr(interfaceLocalAddr.String()))),
		uintptr(unsafe.Pointer(syscall.StringBytePtr(dnsString))),
		uintptr(op))

	return checkDefaultAPIResp(retval, err)
}

func checkDefaultAPIResp(retval uintptr, err error) error {
	if err != syscall.Errno(0) {
		return err
	}
	if retval != 0 {
		return fmt.Errorf("DNS change error: 0x%X", retval)
	}
	return nil
}

// last custom-DNS info which was enabled
var (
	_lastDNS net.IP
)

func catchPanic(err *error) {
	if r := recover(); r != nil {
		log.Error("PANIC (recovered): ", r)
		if e, ok := r.(error); ok {
			*err = e
		} else {
			*err = errors.New(fmt.Sprint(r))
		}
	}
}

// Pause - (on vpn paused) temporary restore OS default DNS parameters
func implPause() error {
	// Not in use for Windows implementation
	// In paused state we are simply switching to the main network interface (to default routes)

	// TODO: in case of custom DNS from local network - necessary to remove custom-DNS configuration from main (non-ivpn) network interface ???

	return nil
}

// Resume - (on vpn resumed) set VPN-defined DNS parameters
func implResume() error {
	// Not in use for Windows implementation
	// In paused state we are simply switching to the main network interface (to default routes)

	// TODO: in case of custom DNS from local network - necessary to add (restore) custom-DNS configuration to main (non-ivpn) network interface ???

	return nil
}

func implSetManual(addr net.IP, localInterfaceIP net.IP) (err error) {
	defer catchPanic(&err)

	if addr.Equal(_lastDNS) {
		return nil
	}

	if _lastDNS != nil {
		// if there was defined DNS - remove it from non-VPN inerfaces (if necessary)
		// (skipping VPN interface, because its data will be owerwrited)
		if err := implDeleteManual(nil); err != nil {
			return fmt.Errorf("Failed to set DNS: %w", err)
		}
	}

	// non-VPN interfaces to update (if DNS located in local network)
	notVpnIfcMACToUpdate, err := getMACAddrByIPinNetwork(addr, localInterfaceIP)

	if localInterfaceIP == nil && len(notVpnIfcMACToUpdate) <= 0 {
		return nil
	}

	start := time.Now()
	log.Info("Changing DNS to ", addr, " ...")
	defer func() {
		if err != nil {
			log.Info(fmt.Sprintf("Changing DNS to %s done (%dms) with error: %s", addr.String(), time.Since(start).Milliseconds(), err.Error()))
		} else {
			log.Info(fmt.Sprintf("Changing DNS to %s: done (%dms)", addr.String(), time.Since(start).Milliseconds()))
		}
	}()

	if localInterfaceIP != nil {
		// SET DNS to VPN interface
		if err := fSetDNSByLocalIP(localInterfaceIP, addr, OperationSet); err != nil {
			return fmt.Errorf("failed to set DNS for local interface: %w", err)
		}
	}

	if len(notVpnIfcMACToUpdate) > 0 {
		// ADD DNS to non-VPN interface (if necessary, when DNS is in local network)

		for _, mac := range notVpnIfcMACToUpdate {
			if err := fSetDNSByMAC(mac, addr, OperationAdd); err != nil {
				return fmt.Errorf("failed to set DNS for interface by MAC: %w", err)
			}
		}
	}

	// save last changed DNS address
	_lastDNS = addr

	return nil
}

func implDeleteManual(localInterfaceIP net.IP) (err error) {
	defer catchPanic(&err)

	// non-VPN interfaces to update (if DNS located in local network)
	notVpnIfcMACToUpdate, err := getMACAddrByIPinNetwork(_lastDNS, localInterfaceIP)

	if localInterfaceIP == nil && len(notVpnIfcMACToUpdate) <= 0 {
		return nil
	}

	start := time.Now()
	log.Info("Restoring default DNS...")
	defer func() {
		if err != nil {
			log.Info(fmt.Sprintf("Restoring default DNS done (%dms) with error: %s", time.Since(start).Milliseconds(), err.Error()))
		} else {
			log.Info(fmt.Sprintf("Restoring default DNS: done (%dms)", time.Since(start).Milliseconds()))
		}
	}()

	if localInterfaceIP != nil {
		// RESET DNS for VPN interface
		if err := fSetDNSByLocalIP(localInterfaceIP, net.IPv4zero, OperationSet); err != nil {
			return fmt.Errorf("failed to reset DNS for local interface: %w", err)
		}
	}

	if len(notVpnIfcMACToUpdate) > 0 {
		// REMOVE DNS from non-VPN interface (if necessary, when DNS is in local network)
		for _, mac := range notVpnIfcMACToUpdate {
			if err := fSetDNSByMAC(mac, _lastDNS, OperationDel); err != nil {
				return fmt.Errorf("failed to reset DNS for interface by MAC: %w", err)
			}
		}
	}

	_lastDNS = nil

	return nil
}

// getMACAddrByIPinNetwork - get hardware addresses (MAC) of the network interfaces to which belongs an IP address (MAC of interface which is in same network as 'addr')
// 		addr - IP address from local network (which can be accessed by interface different to VPN interface)
//		localAddrToSkip - local IP of interface which can be excluded from output (e.g. VPN interface)
func getMACAddrByIPinNetwork(addr net.IP, localAddrToSkip net.IP) (ret []net.HardwareAddr, err error) {
	if addr == nil {
		return ret, nil
	}

	// get interfaces which must be midified by new DNS value
	networks, err := netinfo.GetAllLocalV4Addresses()
	if err != nil {
		return nil, fmt.Errorf("error receiving local V4 addresses : %w", err)
	}

	for _, network := range networks {

		if network.IP.Equal(localAddrToSkip) {
			continue
		}

		if network.Contains(addr) { // 'addr' is in 'network'
			// trying to get MAC address of the network which must be updated
			infs, err := netinfo.InterfaceByIPAddr(network.IP)
			if err != nil {
				log.Error("Failed to get interface for address ", network.IP, ":", err)
				continue
			}

			if infs == nil || infs.HardwareAddr == nil {
				continue
			}

			ret = append(ret, infs.HardwareAddr)
		}
	}

	return ret, nil
}
