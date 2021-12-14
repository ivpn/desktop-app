//+build linux

//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2021 Privatus Limited.
//
//  This file is part of the Daemon for IVPN Client Desktop.
//
//  The Daemon for IVPN Client Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The Daemon for IVPN Client Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the Daemon for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
//

package oshelpers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	_ "image/jpeg"
	_ "image/png"

	"github.com/ivpn/desktop-app/daemon/oshelpers/linux/applist"
	"github.com/ivpn/desktop-app/daemon/oshelpers/linux/icotheme"
)

type extraArgsGetInstalledApps struct {
	EnvVar_XDG_CURRENT_DESKTOP string
	EnvVar_XDG_DATA_DIRS       string
	EnvVar_HOME                string
	IconsTheme                 string
}

// Specification:
// https://specifications.freedesktop.org/desktop-entry-spec/desktop-entry-spec-latest.html
func implGetInstalledApps(extraArgsJSON string) ([]AppInfo, error) {
	XDG_DATA_DIRS := ""
	XDG_CURRENT_DESKTOP := ""
	HOME := ""
	IconsThemeName := ""

	// parse argument
	var extraArgs extraArgsGetInstalledApps
	if len(extraArgsJSON) > 0 {
		if err := json.Unmarshal([]byte(extraArgsJSON), &extraArgs); err == nil {
			XDG_DATA_DIRS = extraArgs.EnvVar_XDG_DATA_DIRS
			HOME = extraArgs.EnvVar_HOME
			XDG_CURRENT_DESKTOP = extraArgs.EnvVar_XDG_CURRENT_DESKTOP
			IconsThemeName = extraArgs.IconsTheme // Yaru
		}
	}

	// read info about all installed apps
	entries := applist.GetAppsList(XDG_DATA_DIRS, XDG_CURRENT_DESKTOP, HOME)

	// Initialize icons theme
	theme, err := icotheme.GetTheme(IconsThemeName, HOME, XDG_DATA_DIRS)
	if err != nil {
		log.Warning("unable to read icons theme: ", err)
	}

	// converting results to AppInfo
	retValues := make([]AppInfo, 0, len(entries))
	for _, e := range entries {
		if e.Name == "IVPN" {
			continue
		}

		base64Img := ""
		if theme.IsInitialized() {
			file, err := theme.FindIcon(e.Icon, []int{32, 48, 24, 64, 22, 128, 256}, []string{"svg", "png"})
			if err == nil {
				if ret, err := readImgToBase64(file); err == nil {
					base64Img = ret
				}
			}
		}
		app := AppInfo{AppName: e.Name, AppBinaryPath: e.Exec, AppIcon: base64Img}
		retValues = append(retValues, app)
	}

	return retValues, nil
}

func implGetFunc_BinaryIconBase64() func(binaryPath string) (icon string, err error) {
	return nil
}

func readImgToBase64(imagePath string) (string, error) {
	if len(imagePath) <= 0 {
		return "", fmt.Errorf("image path is empty")
	}
	// Read the entire file into a byte slice
	bytes, err := ioutil.ReadFile(imagePath)
	if err != nil {
		return "", err
	}

	var base64Encoding string

	// Determine the content type of the image file
	mimeType := http.DetectContentType(bytes)

	// Prepend the appropriate URI scheme header depending
	// on the MIME type
	switch mimeType {
	case "image/jpeg":
		base64Encoding += "data:image/jpeg;base64,"
	case "image/png":
		base64Encoding += "data:image/png;base64,"
	default:
		if strings.ToLower(path.Ext(imagePath)) == ".svg" {
			base64Encoding += "data:image/svg+xml;base64,"
		} else {
			log.Debug("Unsupported format: " + mimeType + " => " + imagePath)
			return "", fmt.Errorf("unsupported image type")
		}
	}

	// Append the base64 encoded output
	base64Encoding += base64.StdEncoding.EncodeToString(bytes)

	return base64Encoding, nil
}
