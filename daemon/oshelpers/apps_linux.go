//go:build linux
// +build linux

//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 IVPN Limited.
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
	"net/http"
	"os"
	"path"
	"path/filepath"
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
	excludeApps := make(map[string]struct{}, 0)
	excludeApps["/usr/bin/gnome-terminal"] = struct{}{} // Terminal is not possible to run in ST
	entries := applist.GetAppsList(XDG_DATA_DIRS, XDG_CURRENT_DESKTOP, HOME, excludeApps)

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

	// open file for reading
	file, err := os.Open(filepath.Clean(imagePath))
	if err != nil {
		return "", err
	}
	defer file.Close()

	finfo, err := file.Stat()
	if err != nil {
		return "", err
	}

	// Ensure the file has read permissions for everyone (check permissions: ---|---|r--)
	if finfo.Mode()&(1<<2) == 0 {
		return "", fmt.Errorf("file '%s' is not allowed to read for everyone", imagePath)
	}

	// Check required buffer size
	var size int
	size64 := finfo.Size()
	if int64(int(size64)) != size64 {
		return "", fmt.Errorf("image size is too big")
	}
	size = int(size64) + 1
	if size < 512 {
		size = 512 // If a file claims a small size, read at least 512 bytes.
	}

	// Read the entire file into a byte slice
	bytes := make([]byte, size)
	if _, err = file.Read(bytes); err != nil {
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
			// remove zero byte from the end (if exists)
			bytes = []byte(strings.TrimSuffix(string(bytes), string([]byte{0})))

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
