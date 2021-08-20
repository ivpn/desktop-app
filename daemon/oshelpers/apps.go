package oshelpers

import "fmt"

// GetInstalledApps returns a list of installed applications on the system
// Return format:
//	map[binaryPath]description
// 	Where 'description' has format: [<app group>/]<AppName>.
// 		Description example: "Git/Git GUI" or "Firefox"
func GetInstalledApps() (apps map[string]string, err error) {
	defer func() {
		if r := recover(); r != nil {
			apps = nil
			if theErr, ok := r.(error); ok {
				err = fmt.Errorf("PANIC on GetInstalledApps() [recovered] : %w", theErr)
			} else {
				err = fmt.Errorf("PANIC on GetInstalledApps() [recovered] ")
			}
		}
	}()

	return implGetInstalledApps()
}
