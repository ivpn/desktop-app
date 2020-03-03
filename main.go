package main

import (
	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/service/platform"
	"github.com/ivpn/desktop-app-daemon/version"
)

func main() {
	logger.Init(platform.LogFile())
	logger.Info("version:" + version.GetFullVersion())
	Launch()
}
