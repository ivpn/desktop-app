package netchange

import (
	"ivpn/daemon/logger"
	"net"
	"time"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("NetChD")
}

const delayBeforeNotify = time.Second * 3

// Detector - object is detecting routing changes on a PC.
// To avoid multiple notifications of multiple changes - the  'DelayBeforeNotify' is in use
// (notification will occur after 'DelayBeforeNotify' elapsed since last detected change)
type Detector struct {
	delayBeforeNotify       time.Duration
	timerNotifyAfterDelay   *time.Timer
	routingChangeNotifyChan chan<- struct{}
	interfaceToProtect      *net.Interface

	// Must be implemeted (AND USED) in correspond file for concrete platform. Must contain platform-specified properties (or can be empty struct)
	props osSpecificProperties
}

// Create - create new network change detector
// 'routingChangeChan' is a notification channel
func Create() *Detector {
	// initialize detector object
	detector := &Detector{delayBeforeNotify: delayBeforeNotify}

	// initialize 'delay'-timer
	timer := time.AfterFunc(0, func() {
		detector.notifyRoutingChange()
	})
	// ensure timer is stopped (no chnages detected for now)
	timer.Stop()

	// save timer
	detector.timerNotifyAfterDelay = timer

	return detector
}

// Start - start route change detector (asynchronous)
func (d *Detector) Start(routingChangeChan chan<- struct{}, currentDefaultInterface *net.Interface) {
	// Ensure that detector is stopped
	d.Stop()

	// set notification channel (it is important to do it after we are ensure that timer is stopped)
	d.routingChangeNotifyChan = routingChangeChan

	// save current default interface
	d.interfaceToProtect = currentDefaultInterface

	// method should be implemented in platform-specific file
	go d.doStart()
}

// Stop - stop route change detector
func (d *Detector) Stop() {
	// stop timer
	d.timerNotifyAfterDelay.Stop()
	// method should be implemented in platform-specific file
	d.doStop()
}

// DelayBeforeNotify - To avoid multiple notifications of multiple changes - the  'DelayBeforeNotify' is in use
// (notification will occur after 'DelayBeforeNotify' elapsed since last detected change)
func (d *Detector) DelayBeforeNotify() time.Duration {
	return d.delayBeforeNotify
}

// must be called when routing change detected (called from platform-specific sources)
func (d *Detector) routingChangeDetected() {
	d.timerNotifyAfterDelay.Reset(d.DelayBeforeNotify())
}

func (d *Detector) notifyRoutingChange() {
	if d.routingChangeNotifyChan == nil {
		return
	}

	var changed bool = false
	var err error = nil

	// method should be implemented in platform-specific file
	// It shell compare current routing configuration with configuration which was when 'doStart()' called
	if changed, err = d.isRoutingChanged(); err != nil {
		return
	}

	if changed {
		select {
		case d.routingChangeNotifyChan <- struct{}{}:
			log.Info("Route change detected. Internet traffic is no longer being routed through the VPN.")
			// notified
		default:
			// channel if full
		}
	}
}
