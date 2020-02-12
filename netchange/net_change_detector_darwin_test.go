package netchange_test

import (
	"fmt"
	"net"
	"os/exec"
	"testing"
	"time"

	"github.com/ivpn/desktop-app-daemon/netchange"
	"github.com/ivpn/desktop-app-daemon/netinfo"
)

func TestDetector(t *testing.T) {
	eventsSenderDone := make(chan struct{})
	readerDone := make(chan struct{})
	notifierChan := make(chan struct{}, 1)

	lastEvtTime := time.Now()
	eventsReceived := 0

	d := netchange.Create()
	fmt.Println("Min interval: ", d.DelayBeforeNotify())

	// events reader
	go func() {
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

	// start detector
	d.Start(notifierChan, anotherInf)
	defer d.Stop()

	// wait some time to give detector fully start
	time.Sleep(time.Second * 2)

	eventsExpected := 0
	// start sender
	go func() {
		defer func() { eventsSenderDone <- struct{}{} }()
		delay := d.DelayBeforeNotify()

		exec.Command("route", "-n", "add", "88.88.88.88", "99.99.99.99").Run()
		time.Sleep(delay)
		fmt.Println("\tONE detection must occur")
		eventsExpected++

		exec.Command("route", "-n", "delete", "88.88.88.88", "99.99.99.99").Run()
		time.Sleep(delay)
		fmt.Println("\tONE detection must occur")
		eventsExpected++

		exec.Command("route", "-n", "add", "88.88.88.88", "99.99.99.99").Run()
		time.Sleep(delay)
		fmt.Println("\tONE detection must occur")
		eventsExpected++

		exec.Command("route", "-n", "delete", "88.88.88.88", "99.99.99.99").Run()
		time.Sleep(delay)
		fmt.Println("\tONE detection must occur")
		eventsExpected++

		exec.Command("route", "-n", "add", "88.88.88.88", "99.99.99.99").Run()
		time.Sleep(delay / 5)
		exec.Command("route", "-n", "delete", "88.88.88.88", "99.99.99.99").Run()
		time.Sleep(delay / 5)
		exec.Command("route", "-n", "add", "88.88.88.88", "99.99.99.99").Run()
		time.Sleep(delay / 5)
		exec.Command("route", "-n", "delete", "88.88.88.88", "99.99.99.99").Run()
		time.Sleep(delay / 5)
		fmt.Println("\tONE detection must occur")
		eventsExpected++

		time.Sleep(delay)

		d.Stop()

		exec.Command("route", "-n", "add", "88.88.88.88", "99.99.99.99").Run()
		time.Sleep(delay / 5)
		exec.Command("route", "-n", "delete", "88.88.88.88", "99.99.99.99").Run()
		// no detections should be here

	}()

	// wait to sender stop
	<-eventsSenderDone

	// stop receiver
	readerDone <- struct{}{}

	if eventsReceived < eventsExpected {
		t.Error("Missing some route change notifications")
	} else if eventsReceived > eventsExpected {
		t.Error("Received route change notifications count more than expected")
	}
}
