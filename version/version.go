package version

import (
	"fmt"
	"strings"
)

// Package provides information about current binary
// In order to integrate version info, use '-ldflags' with a build.
// Example:
// 	go build -ldflags "-X github.com/ivpn/desktop-app-daemon/version._version=v1.0.0 -X github.com/ivpn/desktop-app-daemon/version._time=$(date)"
//
// Example 2:
// 	VERSION="v1.0.0"
//	DATE="$(date "+%Y-%m-%d")"
//	COMMIT="$(git rev-list -1 HEAD)"
//	go build -ldflags "-X github.com/ivpn/desktop-app-daemon/version._version=$VERSION -X github.com/ivpn/desktop-app-daemon/version._commit=$COMMIT -X github.com/ivpn/desktop-app-daemon/version._time=$DATE"

// // application version
var _version string

// build date
var _time string

// developer
var _user string

// Git commit
var _commit string

// Version returns version string
func Version() string {
	return _version
}

// GetFullVersion returns version info string
func GetFullVersion() string {
	verInfo := ""

	if len(_time) > 0 {
		verInfo = verInfo + fmt.Sprintf("date:%s ", _time)
	}
	if len(_user) > 0 {
		verInfo = verInfo + fmt.Sprintf("user:%s ", _user)
	}
	if len(_commit) > 0 {
		verInfo = verInfo + fmt.Sprintf("commit:%s", _commit)
	}

	ret := _version
	if len(verInfo) > 0 {
		if len(ret) > 0 {
			ret += " "
		}
		ret += "(" + strings.TrimSpace(verInfo) + ")"
	}

	if len(ret) == 0 {
		ret = "<version unknown>"
	}

	return ret
}
