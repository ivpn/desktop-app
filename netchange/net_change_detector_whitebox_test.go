package netchange

import (
	"fmt"
	"ivpn/daemon/netinfo"
	"net"
	"testing"
	"time"
)

func TestDetectorDelay(t *testing.T) {
	eventsSenderDone := make(chan struct{})
	readerDone := make(chan struct{})
	notifierChan := make(chan struct{}, 1)

	lastEvtTime := time.Now()
	eventsReceived := 0

	d := Create()
	// change default delay to increase test execution
	d.delayBeforeNotify = time.Second / 10
	fmt.Println("Min interval: ", d.DelayBeforeNotify())

	// we have to simulate another default interface to protect
	defaultInf, _ := netinfo.DefaultRoutingInterface()
	var anotherInf *net.Interface
	ifaces, _ := net.Interfaces()
	for _, ifs := range ifaces {
		if &ifs != defaultInf {
			anotherInf = &ifs
			break
		}
	}

	// events reader
	go func() {
		d.routingChangeNotifyChan = notifierChan
		for {
			select {
			case <-notifierChan:
				timediff := time.Since(lastEvtTime)
				fmt.Println("Notified! Diff:", timediff)
				if timediff < d.DelayBeforeNotify() {
					t.Error("Roure change events received too fast:", timediff, " ( expected >=", d.DelayBeforeNotify(), ")")
				}
				eventsReceived++
			case <-readerDone:
				return
			}
		}
	}()

	// start sender
	go func() {
		defer func() { eventsSenderDone <- struct{}{} }()

		d.interfaceToProtect = anotherInf

		// FIRST NOTIFICATION (duration ~2delays):
		for i := 0; i < 30; i++ {
			d.routingChangeDetected()
		}
		time.Sleep(d.DelayBeforeNotify() - d.DelayBeforeNotify()/10)
		for i := 0; i < 100; i++ {
			d.routingChangeDetected()
		}
		time.Sleep(d.DelayBeforeNotify() + d.DelayBeforeNotify()/10)
		fmt.Println("\tONE detection must be already occured")

		// SECOND NOTIFICATION (duration ~2delays):
		for i := 0; i < 5; i++ {
			d.routingChangeDetected()
			time.Sleep(d.DelayBeforeNotify() / 5)
		}
		time.Sleep(d.DelayBeforeNotify())
		fmt.Println("\tONE detection must be already occured")

		// no notifications should be here (duration ~2delays):
		time.Sleep(d.DelayBeforeNotify() * 2)

		// NO NOTIFICATIONS:
		d.interfaceToProtect = defaultInf
		for i := 0; i < 5; i++ {
			d.routingChangeDetected()
			time.Sleep(d.DelayBeforeNotify() / 5)
		}
		time.Sleep(d.DelayBeforeNotify())

		// THIRD NOTIFICATION:
		d.interfaceToProtect = anotherInf
		d.routingChangeDetected()
		time.Sleep(d.DelayBeforeNotify() * 2)
		fmt.Println("\tONE detection must be already occured")
	}()

	// wait to sender stop
	<-eventsSenderDone
	time.Sleep(d.DelayBeforeNotify() * 8)

	// stop receiver
	readerDone <- struct{}{}

	if eventsReceived < 3 {
		t.Error("Missing some route change notifications")
	} else if eventsReceived > 3 {
		t.Error("Received route change notifications count more than expected")
	}
}
