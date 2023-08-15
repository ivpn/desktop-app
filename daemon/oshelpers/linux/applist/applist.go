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

package applist

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
)

type DesktopEntry struct {
	Name string
	Icon string
	Exec string
}

// GetAppsList returns list of DesktopEntry with an information about installed apps in the system
// Arguments:
//
//	vXDG_DATA_DIRS		- environment variable $XDG_DATA_DIRS (e.g. '/usr/share/ubuntu:/usr/local/share/:/usr/share/:/var/lib/snapd/desktop')
//	vXDG_CURRENT_DESKTOP- environment variable $XDG_CURRENT_DESKTOP (e.g. 'ubuntu:GNOME')
//	vHOME				- environment variable $HOME (e.g. '/home/user')
//
// Specification:
// https://specifications.freedesktop.org/desktop-entry-spec/desktop-entry-spec-latest.html
func GetAppsList(evXDG_DATA_DIRS string, evXDG_CURRENT_DESKTOP string, evHOME string, excludeApps map[string]struct{}) []DesktopEntry {
	var XDG_DATA_DIRS []string
	var XDG_CURRENT_DESKTOP = make(map[string]struct{})
	var HOME = ""

	// parse arguments
	HOME = evHOME
	XDG_DATA_DIRS = strings.Split(evXDG_DATA_DIRS, ":")
	if len(XDG_DATA_DIRS) == 1 && XDG_DATA_DIRS[0] == "" {
		XDG_DATA_DIRS = []string{}
	}

	for _, deskval := range strings.Split(evXDG_CURRENT_DESKTOP, ":") {
		XDG_CURRENT_DESKTOP[deskval] = struct{}{}
	}
	if len(XDG_DATA_DIRS) == 0 {
		XDG_DATA_DIRS = append(XDG_DATA_DIRS, "/usr/local/share/")
		XDG_DATA_DIRS = append(XDG_DATA_DIRS, "/usr/share/")
		XDG_DATA_DIRS = append(XDG_DATA_DIRS, "/var/lib/snapd/desktop/")
	}
	if len(XDG_CURRENT_DESKTOP) == 0 {
		XDG_CURRENT_DESKTOP["ubuntu"] = struct{}{}
		XDG_CURRENT_DESKTOP["GNOME"] = struct{}{}
	}

	// lookup folders
	var lookupFolders []string
	if len(HOME) > 0 {
		lookupFolders = append(lookupFolders, path.Join(HOME, ".local", "share"))
	}
	lookupFolders = append(lookupFolders, XDG_DATA_DIRS...)

	// read all aps info
	var parsedEntries = make(map[string]DesktopEntry)
	for _, dir := range lookupFolders {
		readAppsDirectory(path.Join(dir, "applications"), parsedEntries, XDG_CURRENT_DESKTOP)
	}

	// process each app entry
	regexpBinaryArgs := regexp.MustCompile("(\".*\"|\\S*)(.*)")
	regexpExecSpecialArgs := regexp.MustCompile("([^%])(%[fFuUdDnNickvm])")

	var retValues []DesktopEntry
	for _, e := range parsedEntries {
		// remove special args
		e.Exec = regexpExecSpecialArgs.ReplaceAllString(regexpExecSpecialArgs.ReplaceAllString(e.Exec, "$1"), "$1")
		// resolve escapes
		e.Exec = strings.ReplaceAll(e.Exec, `%%`, `%`)
		e.Exec = strings.ReplaceAll(e.Exec, `\\\\ `, `\\ `)
		e.Exec = strings.ReplaceAll(e.Exec, `\\\\`+"`", `\\`+"`")
		e.Exec = strings.ReplaceAll(e.Exec, `\\\\$`, `\\$`)
		e.Exec = strings.ReplaceAll(e.Exec, `\\\\(`, `\\(`)
		e.Exec = strings.ReplaceAll(e.Exec, `\\\\)`, `\\)`)
		e.Exec = strings.ReplaceAll(e.Exec, `\\\\\`, `\\\`)
		e.Exec = strings.ReplaceAll(e.Exec, `\\\\\\\\`, `\\\\`)

		// get app absolute path and arguments
		cols := regexpBinaryArgs.FindStringSubmatch(e.Exec)
		if len(cols) != 3 {
			continue
		}
		execBin := strings.Trim(cols[1], "\"")
		execBinFull, err := exec.LookPath(execBin)
		if err != nil {
			continue
		}

		// check if the entry is not in excluded apps list
		if _, found := excludeApps[execBinFull]; found {
			continue
		}

		e.Exec = strings.TrimSpace(e.Exec)

		retValues = append(retValues, e)
	}

	return retValues
}

func readAppsDirectory(dirpath string, parsedEntries map[string]DesktopEntry, XDG_CURRENT_DESKTOP map[string]struct{}) error {
	dirEntries, err := os.ReadDir(dirpath)
	if err != nil {
		return err
	}

	for _, dEntry := range dirEntries {
		if dEntry.IsDir() {
			continue
		}

		if !strings.HasSuffix(dEntry.Name(), ".desktop") {
			continue
		}

		if _, ok := parsedEntries[dEntry.Name()]; ok {
			continue // app key with the same ID already exists
		}
		entryPath := path.Join(dirpath, dEntry.Name())
		parsedEntry, err := parseDesktopFile(entryPath, XDG_CURRENT_DESKTOP)
		if err == nil {
			parsedEntries[dEntry.Name()] = parsedEntry
		}
	}

	return nil
}

func parseDesktopFile(filepath string, XDG_CURRENT_DESKTOP map[string]struct{}) (DesktopEntry, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return DesktopEntry{}, err
	}
	defer file.Close()

	keyValueRegexp := regexp.MustCompile("^([A-Za-z0-9-]*) *= *(.*)$")

	vIsDesktopEntry := false

	var ret DesktopEntry
	// read file line-by-line (only "Desktop Entry")
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) <= 0 {
			continue
		}
		if line[0:1] == "#" {
			continue
		}
		if line[0:1] == "[" {
			if vIsDesktopEntry {
				// end of Desktop Entry
				break
			}
			vIsDesktopEntry = strings.EqualFold(line, "[Desktop Entry]")
			continue
		}
		if !vIsDesktopEntry {
			continue
		}

		cols := keyValueRegexp.FindStringSubmatch(line)
		if len(cols) != 3 {
			continue
		}
		key := cols[1]
		val := cols[2]
		if len(key) == 0 || len(val) == 0 {
			continue
		}

		switch key {
		// check fialds indicating we have to skip this file
		case "Type":
			if strings.ToLower(val) != "application" {
				return DesktopEntry{}, fmt.Errorf("entry is not Application type")
			}
		case "Hidden":
			if strings.ToLower(val) == "true" {
				return DesktopEntry{}, fmt.Errorf("app is Hidden")
			}
		case "NoDisplay":
			if strings.ToLower(val) == "true" {
				return DesktopEntry{}, fmt.Errorf("app is NoDisplay")
			}
		case "TryExec":
			if _, err := exec.LookPath(val); err != nil {
				return DesktopEntry{}, fmt.Errorf("the TryExec check failed")
			}
		case "Terminal":
			if strings.ToLower(val) == "true" {
				return DesktopEntry{}, fmt.Errorf("terminal app")
			}

		case "NotShowIn":
			for _, desktop := range strings.Split(val, ";") {
				if _, ok := XDG_CURRENT_DESKTOP[desktop]; ok {
					return DesktopEntry{}, fmt.Errorf("skipped due to NotShowIn")
				}
			}
		case "OnlyShowIn":
			if len(XDG_CURRENT_DESKTOP) > 0 {
				canBeShown := false
				for _, desktop := range strings.Split(val, ";") {
					if _, ok := XDG_CURRENT_DESKTOP[desktop]; ok {
						canBeShown = true
						break
					}
				}
				if !canBeShown {
					return DesktopEntry{}, fmt.Errorf("skipped due to OnlyShowIn")
				}
			}

		// fields we are interesting in

		case "Name":
			ret.Name = val
		case "Icon":
			ret.Icon = val
		case "Exec":
			ret.Exec = val
		}
	}
	return ret, nil
}
