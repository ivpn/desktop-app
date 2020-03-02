package main

import (
	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/service/platform"
)

func main() {
	logger.Init(platform.LogFile())
	Launch()
}
