//+build windows

package oshelpers

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	lnk "github.com/parsiya/golnk"
)

func WinExpandEnvPath(path string) string {
	// match windows-style variables. E.g.: %windir%
	re := regexp.MustCompile("%[^%]+%")
	path = re.ReplaceAllStringFunc(path, func(str string) string {
		return "${" + strings.Trim(str, "%") + "}"
	})
	return os.ExpandEnv(path)
}

func implGetInstalledApps() (map[string]string, error) {

	programData := os.Getenv("PROGRAMDATA")
	appData := os.Getenv("APPDATA")
	programDataSMDir := ""
	appDataSMDir := ""

	excludeStartMenuPaths := make([]string, 0, 2)
	if len(programData) > 0 {
		programDataSMDir = programData + `\Microsoft\Windows\Start Menu\Programs`
		excludeStartMenuPaths = append(excludeStartMenuPaths, strings.ToLower(programDataSMDir+`\startup`))
		excludeStartMenuPaths = append(excludeStartMenuPaths, strings.ToLower(programDataSMDir+`\Administrative Tools`))
	}
	if len(appData) > 0 {
		appDataSMDir = appData + `\Microsoft\Windows\Start Menu\Programs`
		excludeStartMenuPaths = append(excludeStartMenuPaths, strings.ToLower(appDataSMDir+`\startup`))
		excludeStartMenuPaths = append(excludeStartMenuPaths, strings.ToLower(appDataSMDir+`\Administrative Tools`))
	}

	// ignore all binaries from IVPN installation
	excludeBinPath := ""
	if ex, err := os.Executable(); err == nil && len(ex) > 0 {
		excludeBinPath = strings.ToLower(filepath.Dir(ex))
	}

	retMap := make(map[string]string) // [path]description
	walkFunc := func(path string, info os.FileInfo, walkErr error) (err error) {

		defer func() {
			if r := recover(); r != nil {
				errText := ""
				if theErr, ok := r.(error); ok {
					errText = fmt.Sprintf("PANIC [recovered] on implGetInstalledApps() for '%s' : %v", path, theErr)
				} else {
					errText = fmt.Sprintf("PANIC [recovered] on implGetInstalledApps() for '%s'", path)
				}
				log.Error(errText)
			}
		}()

		// Only look for lnk files.
		if filepath.Ext(info.Name()) == ".lnk" {

			// ignore files from 'excludePaths'
			for _, excludePath := range excludeStartMenuPaths {
				curLnkPath := strings.ToLower(path)
				if strings.HasPrefix(curLnkPath, excludePath) {
					return nil
				}
			}

			f, lnkErr := lnk.File(path)

			if lnkErr != nil {
				return nil
			}

			var targetPath = ""
			if f.LinkInfo.LocalBasePath != "" {
				targetPath = f.LinkInfo.LocalBasePath
			}
			if f.LinkInfo.LocalBasePathUnicode != "" {
				targetPath = f.LinkInfo.LocalBasePathUnicode
			}
			if f.StringData.IconLocation != "" {
				targetPath = f.StringData.IconLocation
			}

			// Only look for exe files.
			if targetPath != "" && filepath.Ext(targetPath) == ".exe" {
				baseDir := filepath.Dir(path)

				if strings.EqualFold(baseDir, programDataSMDir) || strings.EqualFold(baseDir, appDataSMDir) {
					baseDir = ""
				} else {
					baseDir = filepath.Base(baseDir) + "/"
				}

				// expand all environment variables in file path
				targetPath = WinExpandEnvPath(targetPath)

				if _, err := os.Stat(targetPath); os.IsNotExist(err) {
					// file not exists
					return nil
				}

				// ignore all binaries from IVPN installation
				if strings.HasPrefix(strings.ToLower(targetPath), excludeBinPath) {
					return nil
				}

				retMap[targetPath] = baseDir + strings.TrimSuffix(info.Name(), ".lnk")
			}
		}

		return nil
	}

	if len(programDataSMDir) > 0 {
		filepath.Walk(programDataSMDir, walkFunc)
	}

	if len(appDataSMDir) > 0 {
		filepath.Walk(appDataSMDir, walkFunc)
	}

	return retMap, nil
}
