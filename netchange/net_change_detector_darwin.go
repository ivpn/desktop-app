package netchange

import (
	"os"
	"strings"
	"syscall"

	"github.com/ivpn/desktop-app-daemon/netinfo"

	"golang.org/x/net/route"
)

// structure contains properties required for for macOS implementation
type osSpecificProperties struct {
	socket int
}

func (d *Detector) isRoutingChanged() (bool, error) {
	if d.interfaceToProtect == nil {
		log.Error("failed to check route change. Initial interface not defined")
		return false, nil
	}

	ifc, err := netinfo.DefaultRoutingInterface()

	if err != nil {
		log.Error("Failed to check route change:", err)
		return false, err
	}

	if strings.Compare(ifc.Name, d.interfaceToProtect.Name) != 0 {
		return true, nil
	}

	return false, nil
}

func (d *Detector) doStart() {
	sock, err := syscall.Socket(syscall.AF_ROUTE, syscall.SOCK_RAW, syscall.AF_UNSPEC)
	if err != nil {
		log.Error("Failed to start route change detector:", err)
		return
	}
	d.props.socket = sock

	log.Info("Route change detector started")
	defer func() {
		log.Info("Route change detector stopped")
		d.doStop()
	}()

	// Loop waiting for messages.
	b := make([]byte, os.Getpagesize())
	for {
		nr, err := syscall.Read(d.props.socket, b)
		if err != nil {
			if d.props.socket == 0 {
				break
			}
			log.Error("Route change detector (error on socket read):", err)
			return
		}

		messages, err := route.ParseRIB(0, b[:nr])
		if err != nil {
			continue
		}

		for _, msg := range messages {
			switch rmsg := msg.(type) {
			case *route.RouteMessage:
				switch rmsg.Type {
				case syscall.RTM_ADD, syscall.RTM_CHANGE, syscall.RTM_DELETE:
					d.routingChangeDetected()
				}
			}
		}
	}
}

func (d *Detector) doStop() {
	s := d.props.socket
	d.props.socket = 0
	if s != 0 {
		syscall.Close(s)
	}
}
