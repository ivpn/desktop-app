package netinfo

import (
	"fmt"
	"testing"
)

func TestPrintRoutes(t *testing.T) {
	routes, err := getWindowsIPv4Routes()
	if err != nil {
		t.Fail()
	}
	for _, r := range routes {
		fmt.Println(r)
	}
}

func TestGetDefaultGateway(t *testing.T) {
	gw, err := doDefaultGatewayIP()
	if err != nil {
		t.Fail()
	}

	fmt.Println(gw)
}
