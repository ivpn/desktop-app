package oshelpers

import (
	"fmt"

	"github.com/ivpn/desktop-app/daemon/logger"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("oshlpr")
}

type AppInfo struct {
	// Application description: [<AppGroup>/]<AppName>.
	// Example 1: "Git/Git GUI"
	// 		AppName  = "Git GUI"
	// 		AppGroup = "Git"
	// Example 2: "Firefox"
	// 		AppName  = "Firefox"
	// 		AppGroup = null
	AppName       string
	AppGroup      string // optional
	AppBinaryPath string // absolute path to application binary
	AppIcon       string // base64 png icon of the executable binary
}

// GetInstalledApps returns a list of installed applications on the system
// Parameters:
// 	extraArgsJSON - (optional) Platform-depended: extra parameters (in JSON)
// 	For Windows:
//		{ "WindowsEnvAppdata": "..." }
// 		Applicable only for Windows: APPDATA environment variable
// 		Needed to know path of current user's (not root) StartMenu folder location
func GetInstalledApps(extraArgsJSON string) (apps []AppInfo, err error) {
	defer func() {
		if r := recover(); r != nil {
			apps = nil
			if theErr, ok := r.(error); ok {
				err = fmt.Errorf("PANIC on GetInstalledApps() [recovered] : %w", theErr)
			} else {
				err = fmt.Errorf("PANIC on GetInstalledApps() [recovered] ")
			}
			log.Error(err)
		}
	}()

	return implGetInstalledApps(extraArgsJSON)
}

func GetBinaryIconBase64Png(binaryPath string) (icon string, err error) {
	defer func() {
		if r := recover(); r != nil {
			icon = ""
			if theErr, ok := r.(error); ok {
				err = fmt.Errorf("PANIC on GetBinaryBase64PngIcon() [recovered] : %w", theErr)
			} else {
				err = fmt.Errorf("PANIC on GetBinaryIconBase64Png() [recovered] ")
			}
			log.Error(err)
		}
	}()

	return implGetBinaryIconBase64Png(binaryPath)
}
