//+build windows

package oshelpers

import (
	"os"
	"path/filepath"
	"strings"

	lnk "github.com/parsiya/golnk"
)

func implGetInstalledApps() (map[string]string, error) {

	programData := os.Getenv("PROGRAMDATA")
	appData := os.Getenv("APPDATA")
	programDataSMDir := ""
	appDataSMDir := ""

	if len(programData) > 0 {
		programDataSMDir = programData + `\Microsoft\Windows\Start Menu\Programs`
	}
	if len(appData) > 0 {
		appDataSMDir = appData + `\Microsoft\Windows\Start Menu\Programs`
	}

	retMap := make(map[string]string) // [path]description
	walkFunc := func(path string, info os.FileInfo, walkErr error) error {

		// Only look for lnk files.
		if filepath.Ext(info.Name()) == ".lnk" {
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

			// Only look for exe files.
			if targetPath != "" && filepath.Ext(targetPath) == ".exe" {
				baseDir := filepath.Dir(path)

				if strings.EqualFold(baseDir, programDataSMDir) || strings.EqualFold(baseDir, appDataSMDir) {
					baseDir = ""
				} else {
					baseDir = filepath.Base(baseDir) + "/"
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
