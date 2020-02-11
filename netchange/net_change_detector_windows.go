package netchange

import (
	"errors"
	"fmt"
	"ivpn/daemon/netinfo"
	"ivpn/daemon/oshelpers/windows/iphlpapi"
	"ivpn/daemon/oshelpers/windows/kernel32"
	"ivpn/daemon/oshelpers/windows/winsock2"
	"net"
	"syscall"
)

// structure contains properties required for for macOS implementation
type osSpecificProperties struct {
	overlapped syscall.Overlapped
}

func (d *Detector) isRoutingChanged() (bool, error) {
	infToProtect := d.interfaceToProtect
	if infToProtect == nil {
		err := errors.New("failed to check route change. Initial interface not defined")
		log.Error(err)
		return false, nil
	}

	// define IP addresses to which the default route will be checked
	ipToCheckRoute := []net.IP{net.IPv4(1, 1, 1, 1), net.IPv4(8, 8, 8, 8)}

	var mib iphlpapi.APIMibIPForwardRow

	// check default route for each IP
	for _, ip := range ipToCheckRoute {
		err := iphlpapi.APIGetBestRoute(ip, net.IPv4(0, 0, 0, 0), &mib)
		if err != nil {
			log.Error("Failed to check route change:", err)
			return false, err
		}

		// check if the interface indexes are same
		if mib.DwForwardIfIndex != uint32(infToProtect.Index) {

			activInterfaceInfo := fmt.Sprintf("#%d", mib.DwForwardIfIndex)
			if inf, err := netinfo.GetInterfaceByIndex(int(mib.DwForwardIfIndex)); err == nil && inf != nil {
				activInterfaceInfo = inf.Name
			}
			log.Info(fmt.Sprintf("Routing change detected. Expected route over '%s'; current route '%s'", infToProtect.Name, activInterfaceInfo))

			return true, nil
		}
	}

	return false, nil
}

func (d *Detector) doStart() {

	log.Info("Route change detector started")
	defer func() {
		log.Info("Route change detector stopped")
	}()

	var handle syscall.Handle

	d.props.overlapped = syscall.Overlapped{}
	d.props.overlapped.HEvent, _ = winsock2.WSACreateEvent()

	for {
		if d.props.overlapped.HEvent == 0 {
			break
		}

		// register route change handler
		err := iphlpapi.APINotifyRouteChange(&handle, &d.props.overlapped)
		if err != nil {
			log.Error(err)
			return
		}

		evtHandle := d.props.overlapped.HEvent
		if evtHandle == 0 {
			return
		}

		_, err = syscall.WaitForSingleObject(evtHandle, syscall.INFINITE)
		if err != nil {
			log.Error(err)
			return
		}

		if d.props.overlapped.HEvent == 0 {
			break
		}

		// notify about routing change
		d.notifyRoutingChange()
	}
}

func (d *Detector) doStop() {
	overlapped := d.props.overlapped

	if overlapped.HEvent != 0 {
		// do not start new route change
		d.props.overlapped.HEvent = 0

		_, err := kernel32.SetEvent(overlapped.HEvent)
		if err != nil {
			log.Error("Failed to stop route change detection (SetEvent 1): ", err)
		}

		// stop route change detection
		err = iphlpapi.CancelIPChangeNotify(&overlapped)
		if err != nil {
			log.Error("Failed to stop route change detection (CancelIPChangeNotify):", err)
		}

		_, err = kernel32.SetEvent(overlapped.HEvent)
		if err != nil {
			log.Error("Failed to stop route change detection (SetEvent 2):", err)
		}
	}
}
