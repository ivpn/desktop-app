//
//  IVPN command line interface (CLI)
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
//
//  This file is part of the IVPN command line interface.
//
//  The IVPN command line interface is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The IVPN command line interface is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the IVPN command line interface. If not, see <https://www.gnu.org/licenses/>.
//

package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ivpn/desktop-app/cli/flags"
	service_types "github.com/ivpn/desktop-app/daemon/protocol/types"
	"github.com/ivpn/desktop-app/daemon/service/platform"
)

type CmdLogs struct {
	flags.CmdInfo
	show    bool
	enable  bool
	disable bool
}

func (c *CmdLogs) Init() {
	c.Initialize("logs", "Logging management")
	c.BoolVar(&c.show, "show", false, "(default) Show logs")
	c.BoolVar(&c.enable, "on", false, "Enable logging")
	c.BoolVar(&c.disable, "off", false, "Disable logging")
}
func (c *CmdLogs) Run() error {
	if c.enable && c.disable {
		return flags.BadParameter{}
	}

	var err error
	if c.enable {
		err = c.setSetLogging(true)
	} else if c.disable {
		err = c.setSetLogging(false)
	}

	if err != nil || c.enable || c.disable {
		return err
	}
	return c.doShow()
}

func (c *CmdLogs) setSetLogging(enable bool) error {
	if enable {
		return _proto.SetPreferences(string(service_types.Prefs_IsEnableLogging), "true")
	}
	return _proto.SetPreferences(string(service_types.Prefs_IsEnableLogging), "false")
}

func (c *CmdLogs) doShow() error {

	isPartOfFile := false
	isSomethingPrinted := false

	fname := platform.LogFile()
	file, err := os.Open(filepath.Clean(fname))
	if err != nil {
		return err
	}
	defer func() {
		file.Close()
		if isSomethingPrinted {
			fmt.Println("##############")
		}
		if isPartOfFile {
			fmt.Println("Printed the last part of the log.")
		}
		fmt.Println("Log file:", fname)
	}()

	stat, err := os.Stat(fname)
	if err != nil {
		return err
	}
	size := stat.Size()

	maxBytesToRead := int64(60 * 50)
	if size > maxBytesToRead {
		isPartOfFile = true
		if _, err := file.Seek(-maxBytesToRead, io.SeekEnd); err != nil {
			return err
		}
	}

	buff := make([]byte, maxBytesToRead)
	if _, err := file.Read(buff); err != nil {
		return err
	}

	fmt.Println(string(buff))
	isSomethingPrinted = true

	if isPartOfFile {
		fmt.Println("##############")
		fmt.Println("To view full log, please refer to file:", fname)
	}

	return nil
}
