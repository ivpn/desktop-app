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

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/ivpn/desktop-app/cli/commands"
	"github.com/ivpn/desktop-app/cli/flags"
	"github.com/ivpn/desktop-app/cli/protocol"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/version"
)

// ICommand interface for command line command
type ICommand interface {
	Init()
	Parse(arguments []string) error
	Run() error

	Name() string
	Description() string
	Usage(short bool)
	UsageFormetted(w *tabwriter.Writer, short bool)
}

var (
	_commands []ICommand
)

func addCommand(cmd ICommand) {
	cmd.Init()
	_commands = append(_commands, cmd)
}

func printHeader() {
	fmt.Println("Command-line interface for IVPN client (www.ivpn.net)")
	fmt.Println("version:" + version.GetFullVersion() + "\n")
}

func printUsageAll(short bool) {
	printHeader()
	fmt.Printf("Usage: %s COMMAND [OPTIONS...] [COMMAND_PARAMETER] [-h|-help]\n\n", filepath.Base(os.Args[0]))

	fmt.Println("COMMANDS:")
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	for _, c := range _commands {
		c.UsageFormetted(writer, short)
		if short == false {
			fmt.Fprintln(writer, "\t")
		}
	}
	writer.Flush()

	if short {
		commands.PrintTips([]commands.TipType{commands.TipHelpCommand, commands.TipHelpFull})
	}
}

func main() {
	// initialize all possible commands
	stateCmd := commands.CmdState{}
	addCommand(&stateCmd)
	addCommand(&commands.CmdConnect{})
	addCommand(&commands.CmdDisconnect{})
	addCommand(&commands.CmdServers{})
	addCommand(&commands.CmdFirewall{})
	if runtime.GOOS == "windows" {
		// Split tunnel functionality is currently only available on Windows
		addCommand(&commands.SplitTun{})
	}
	addCommand(&commands.CmdWireGuard{})
	addCommand(&commands.CmdDns{})
	addCommand(&commands.CmdAntitracker{})
	addCommand(&commands.CmdLogs{})
	addCommand(&commands.CmdLogin{})
	addCommand(&commands.CmdLogout{})
	addCommand(&commands.CmdAccount{})

	if len(os.Args) >= 2 {
		if os.Args[1] == "?" || os.Args[1] == "-?" || os.Args[1] == "-h" || os.Args[1] == "--h" || os.Args[1] == "-help" || os.Args[1] == "--help" {
			if len(os.Args) >= 3 && strings.ToLower(os.Args[2]) == "-full" {
				printUsageAll(false) // detailed commans descriptions
			} else {
				printUsageAll(true) // short commands descriptions
			}
			os.Exit(0)
		}
	}

	// initialize command handler
	port, secret, err := readDaemonPort()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Unable to connect to service: %s\n", err)
		printServStartInstructions()
		os.Exit(1)
	}

	proto := protocol.CreateClient(port, secret)
	if err := proto.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to connect to service : %s\n", err)
		printServStartInstructions()
		os.Exit(1)
	}

	commands.Initialize(proto)

	if len(os.Args) < 2 {
		if err := stateCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "\n%v\n", err)
			os.Exit(1)
		}
		return
	}

	// process command
	isProcessed := false
	for _, c := range _commands {
		if c.Name() == os.Args[1] {
			isProcessed = true
			runCommand(c, os.Args[2:])
			break
		}
	}

	// unknown command
	if isProcessed == false {
		fmt.Fprintf(os.Stderr, "Error. Unexpected command %s\n", os.Args[1])
		printUsageAll(true)
		os.Exit(1)
	}
}

func runCommand(c ICommand, args []string) {
	if err := c.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		if _, ok := err.(flags.BadParameter); ok == true {
			c.Usage(false)
		}
		os.Exit(1)
	}

	if err := c.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		if _, ok := err.(flags.BadParameter); ok == true {
			c.Usage(false)
		}
		os.Exit(1)
	}
}

// read port+secret to be able to connect to a daemon
func readDaemonPort() (port int, secret uint64, err error) {
	file := platform.ServicePortFile()
	if len(file) == 0 {
		return 0, 0, fmt.Errorf("connection-info file not defined")
	}

	if _, err := os.Stat(file); err != nil {
		if os.IsNotExist(err) {
			return 0, 0, fmt.Errorf("please, ensure IVPN daemon is running (connection-info not exists)")
		}
		return 0, 0, fmt.Errorf("connection-info check error: %s", err)
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}

	vars := strings.Split(string(data), ":")
	if len(vars) != 2 {
		return 0, 0, fmt.Errorf("failed to parse connection-info")
	}

	port, err = strconv.Atoi(strings.TrimSpace(vars[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse connection-info: %w", err)
	}

	secret, err = strconv.ParseUint(strings.TrimSpace(vars[1]), 16, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse connection-info: %w", err)
	}

	return port, secret, nil
}
