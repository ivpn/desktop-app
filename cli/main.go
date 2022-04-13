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
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/ivpn/desktop-app/cli/cliplatform"
	"github.com/ivpn/desktop-app/cli/commands"
	"github.com/ivpn/desktop-app/cli/flags"
	"github.com/ivpn/desktop-app/cli/helpers"
	"github.com/ivpn/desktop-app/cli/protocol"
	"github.com/ivpn/desktop-app/daemon/protocol/eap"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/version"
	"golang.org/x/term"
)

// ICommand interface for command line command
type ICommand interface {
	Init()
	Parse(arguments []string) error
	ParseSpecial(arguments []string) (parsedSpecial bool)
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
		if !short {
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
	if cliplatform.IsSplitTunSupported() {
		// Split tunnel functionality is currently only available on Windows
		addCommand(&commands.SplitTun{})
		if cliplatform.IsSplitTunRunsApp() {
			addCommand(&commands.Exclude{})
		}
	}
	addCommand(&commands.CmdWireGuard{})
	addCommand(&commands.CmdDns{})
	addCommand(&commands.CmdAntitracker{})
	addCommand(&commands.CmdLogs{})
	addCommand(&commands.CmdLogin{})
	addCommand(&commands.CmdLogout{})
	addCommand(&commands.CmdAccount{})
	addCommand(&commands.CmdParanoidMode{})

	if len(os.Args) >= 2 {
		arg1 := strings.TrimLeft(strings.ToLower(os.Args[1]), "-")
		arg2 := ""
		if len(os.Args) >= 3 {
			arg2 = strings.TrimLeft(strings.ToLower(os.Args[2]), "-")
		}

		if arg1 == "v" || arg1 == "version" {
			printHeader()
			os.Exit(0)
		}

		if arg1 == "?" || arg1 == "h" || arg1 == "help" {
			if arg2 == "full" {
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

	proto.SetParanoidModeSecretRequestFunc(RequestParanoidModePassword)

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

func RequestParanoidModePassword(c *protocol.Client) (string, error) {
	// for privilaged users: try to read secret directly from file
	if helpers.CheckIsAdmin() {
		sFile := c.GetHelloResponse().ParanoidMode.FilePath
		if len(sFile) > 0 {
			eap := eap.Init(sFile)
			secret, _ := eap.Secret()
			if len(secret) > 0 {
				return secret, nil
			}
		}
	}

	// request secret from user
	fmt.Print("EAP is active. Enter shared secret: ")

	data, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println("")
	if err != nil {
		return "", fmt.Errorf("failed to read EAP shared secret: %s\n", err)
	}
	secret := strings.TrimSpace(string(data))
	if len(secret) <= 0 {
		return "", fmt.Errorf("EAP shared secret not defined")
	}

	return secret, nil
}

func runCommand(c ICommand, args []string) {
	parsedSpecial := c.ParseSpecial(args)
	if !parsedSpecial {
		if err := c.Parse(args); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			if _, ok := err.(flags.BadParameter); ok == true {
				c.Usage(false)
			}
			os.Exit(1)
		}
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

	data, err := ioutil.ReadFile(filepath.Clean(file))
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
