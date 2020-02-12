package obfsproxy_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/ivpn/desktop-app-daemon/obfsproxy"
	"github.com/ivpn/desktop-app-daemon/service/platform"
)

func TestStart(t *testing.T) {
	obfsp := obfsproxy.CreateObfsproxy(platform.ObfsproxyStartScript())

	port, err := obfsp.Start()
	if err != nil {
		fmt.Println("ERROR:", err)
	} else {
		fmt.Println("Started on:", port)
	}

	go func() {
		time.Sleep(time.Second * 5)
		obfsp.Stop()
	}()

	if err := obfsp.Wait(); err != nil {
		fmt.Println("STOP ERROR:", err)
	}
	fmt.Println("STOPED")
}
