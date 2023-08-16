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

package icotheme

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
)

type ThemeDirectory struct {
	Dir   string
	Size  int
	Scale int
}

type Theme struct {
	Name           string
	IndexFile      string
	Inherits       []string
	InheritsParsed []*Theme
	Dirs           map[int]([]ThemeDirectory) // [size]([]dirs)
	IconsBaseDirs  []string
	isInitialized  bool
}

// GetTheme initializes Theme object.
// Parameters:
//
//	themeName 		- name of the theme
//					  (can be obtained by terminal command: "gsettings get org.gnome.desktop.interface icon-theme")
//	HOME 			- environment variavle $HOME (e.g. '/home/user')
//	XDG_DATA_DIRS 	- environment variable $XDG_DATA_DIRS (e.g. '/usr/share/ubuntu:/usr/local/share/:/usr/share/:/var/lib/snapd/desktop')
func GetTheme(themeName string, evHOME string, evXDG_DATA_DIRS string) (Theme, error) {
	// parse arguments
	XDG_DATA_DIRS := strings.Split(evXDG_DATA_DIRS, ":")
	if len(XDG_DATA_DIRS) == 1 && XDG_DATA_DIRS[0] == "" {
		XDG_DATA_DIRS = []string{}
	}
	if len(XDG_DATA_DIRS) == 0 {
		XDG_DATA_DIRS = append(XDG_DATA_DIRS, "/usr/local/share/")
		XDG_DATA_DIRS = append(XDG_DATA_DIRS, "/usr/share/")
		XDG_DATA_DIRS = append(XDG_DATA_DIRS, "/var/lib/snapd/desktop/")
	}
	// read base folders for theme search
	iconsBaseDirs := getIconBaseDirs(evHOME, XDG_DATA_DIRS)

	// Read theme info
	ret, err := readTheme(themeName, iconsBaseDirs)
	if err != nil {
		return ret, err
	}

	// kepp all parsed themes in a map to avoid multiple parsing of the same theme
	processedThemes := make(map[string]*Theme)
	processedThemes[ret.Name] = &ret

	// Read parent themes
	var readParents func(*Theme)
	readParents = func(theme *Theme) {
		if len(theme.InheritsParsed) > 0 {
			return
		}

		for _, name := range theme.Inherits {

			if parsed, ok := processedThemes[name]; ok {
				theme.InheritsParsed = append(theme.InheritsParsed, parsed)
				continue
			}

			t, err := readTheme(name, iconsBaseDirs)
			if err != nil {
				return
			}

			processedThemes[t.Name] = &t
			theme.InheritsParsed = append(theme.InheritsParsed, &t)

			readParents(&t)
		}
	}

	readParents(&ret)

	if _, ok := processedThemes["hicolor"]; !ok {
		hicolorTheme, err := readTheme("hicolor", iconsBaseDirs)
		if err == nil {
			ret.Inherits = append(ret.Inherits, "hicolor")
			ret.InheritsParsed = append(ret.InheritsParsed, &hicolorTheme)
		}
	}

	ret.isInitialized = true
	return ret, nil
}

func readTheme(themeName string, iconsBaseDirs []string) (Theme, error) {
	if len(themeName) <= 0 {
		return Theme{}, fmt.Errorf("theme name not defined")
	}

	themeFile := ""
	for _, d := range iconsBaseDirs {
		f := path.Join(d, themeName, "index.theme")
		if _, err := os.Stat(f); err == nil {
			themeFile = f
			break
		}
	}
	if len(themeFile) <= 0 {
		return Theme{}, fmt.Errorf("theme not found")
	}

	// reading parents info
	file, err := os.Open(themeFile)
	if err != nil {
		return Theme{}, fmt.Errorf("cannot open theme file: '%s'", themeFile)
	}
	defer file.Close()

	var ret Theme
	ret.Inherits = make([]string, 0)
	ret.InheritsParsed = make([]*Theme, 0)
	ret.Dirs = make(map[int][]ThemeDirectory)
	ret.Name = themeName
	ret.IndexFile = themeFile
	ret.IconsBaseDirs = iconsBaseDirs

	getNumValFunc := func(line string) (int, error) {
		vals := strings.Split(line, "=")
		if len(vals) != 2 {
			return 0, fmt.Errorf("failed to parse")
		}
		n, err := strconv.Atoi(vals[1])
		if err != nil {
			return 0, err
		}
		return n, nil
	}

	var curDir ThemeDirectory
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) <= 0 || line[0:1] == "#" {
			continue
		}
		if strings.HasPrefix(line, "Inherits") {
			vals := strings.Split(line, "=")
			if len(vals) != 2 {
				continue
			}
			parents := strings.Split(vals[1], ",")
			for i, p := range parents {
				parents[i] = strings.TrimSpace(p)
			}
			ret.Inherits = parents
			continue
		}

		if line[0:1] == "[" {
			if len(curDir.Dir) > 0 && curDir.Size > 0 {
				if _, ok := ret.Dirs[curDir.Size]; !ok {
					ret.Dirs[curDir.Size] = make([]ThemeDirectory, 0)
				}
				ret.Dirs[curDir.Size] = append(ret.Dirs[curDir.Size], curDir)
			}
			curDir = ThemeDirectory{}

			groupname := strings.Trim(line, "[]")
			if strings.HasPrefix(groupname, "X-") {
				continue
			}
			if groupname != "Icon Theme" {
				curDir.Dir = groupname
			}
			continue
		}

		if len(curDir.Dir) <= 0 {
			continue
		}

		if strings.HasPrefix(line, "Size") {
			n, err := getNumValFunc(line)
			if err == nil {
				curDir.Size = n
			}
			continue
		}
		if strings.HasPrefix(line, "Scale") {
			n, err := getNumValFunc(line)
			if err == nil {
				curDir.Scale = n
			}
		}
	}

	if len(curDir.Dir) > 0 && curDir.Size > 0 {
		ret.Dirs[curDir.Size] = append(ret.Dirs[curDir.Size], curDir)
	}

	return ret, nil
}

func getIconBaseDirs(HOME string, XDG_DATA_DIRS []string) []string {
	ret := make([]string, 0, uint16(len(XDG_DATA_DIRS)+3))
	if len(HOME) > 0 {
		ret = append(ret, path.Join(HOME, ".icons"))
		ret = append(ret, path.Join(HOME, ".local", "share", "icons"))
	}
	for _, d := range XDG_DATA_DIRS {
		ret = append(ret, path.Join(d, "icons"))
	}
	ret = append(ret, "/usr/share/pixmaps/")
	return ret
}

func (c Theme) IsInitialized() bool {
	return c.isInitialized
}

// FindIcon searching for the icon file in the current theme
// Parameters:
//
//	name - icon name
//	desiredSize - expected icon sizes (E.g. [24, 32, 48]; first element - higher priority)
//	desiredSize - expected icon format (E.g. ["png", "svg"]; first element - higher priority)
func (c Theme) FindIcon(name string, desiredSize []int, formats []string) (string, error) {
	if _, err := os.Stat(name); err == nil {
		return name, nil
	}

	if !c.isInitialized {
		return "", fmt.Errorf("theme not initialized")
	}

	if len(name) <= 0 {
		return "", fmt.Errorf("name not defined")
	}
	if len(formats) <= 0 {
		formats = []string{"svg", "png"}
	}

	file, err := c.findIconHelper(name, desiredSize, formats)
	if err == nil {
		return file, nil
	}

	return "", fmt.Errorf("not found")
}

func (c Theme) findIconHelper(name string, desiredSize []int, formats []string) (string, error) {
	// looking in current theme
	for _, basedir := range c.IconsBaseDirs {
		themeDir := path.Join(basedir, c.Name)

		if _, err := os.Stat(themeDir); err != nil {
			continue
		}

		sizesToFind := desiredSize
		if len(sizesToFind) <= 0 {
			// if desired sizes not defined - looking for any size
			for k := range c.Dirs {
				sizesToFind = append(sizesToFind, k)
			}
		}

		for _, fs := range sizesToFind {
			themeSizeDirs, ok := c.Dirs[fs]
			if !ok {
				continue
			}

			for _, sizedir := range themeSizeDirs {
				fullSizeDir := path.Join(themeDir, sizedir.Dir)

				for _, ext := range formats {
					fname := path.Join(fullSizeDir, name+"."+ext)
					if _, err := os.Stat(fname); err == nil {
						return fname, nil
					}
					if strings.HasSuffix(name, "."+ext) {
						fname := path.Join(fullSizeDir, name)
						if _, err := os.Stat(fname); err == nil {
							return fname, nil
						}
					}
				}
			}
		}
	}

	// file not found. Looking in parents
	for _, parent := range c.InheritsParsed {
		file, err := parent.findIconHelper(name, desiredSize, formats)
		if err == nil {
			return file, err
		}
	}

	// file not found: looking in /usr/share/pixmaps/
	for _, ext := range formats {
		fname := path.Join("/usr/share/pixmaps/", name+"."+ext)
		if _, err := os.Stat(fname); err == nil {
			return fname, nil
		}
	}

	return "", fmt.Errorf("not found")
}
