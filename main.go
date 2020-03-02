package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ivpn/desktop-app-cli/protocol"
	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/service/platform"
)

// ICommand interface for command line command
type ICommand interface {
	Init()
	Description() (description string, isHasArguments bool)
	FlagSet() *flag.FlagSet
	Run() error
}

var (
	_commands []ICommand
	_proto    *protocol.Client
)

func addCommand(cmd ICommand) {
	cmd.Init()

	if description, hasArguments := cmd.Description(); hasArguments == false {
		cmd.FlagSet().Usage = func() {
			fmt.Printf("%s - %s\n", cmd.FlagSet().Name(), description)
			fmt.Printf("  Usage of %s: no arguments for this command\n", cmd.FlagSet().Name())
		}
	}

	_commands = append(_commands, cmd)
}

func printHeader() {
	fmt.Println("Command-line interface for IVPN client (www.ivpn.net)")
}

func printUsageAll() {
	printHeader()
	fmt.Printf("Usage: %s <command> [-parameter1 [<argument>] ... -parameterN [<argument>]] [-h|-help] \n", filepath.Base(os.Args[0]))

	fmt.Println("COMANDS:")
	fmt.Println()
	for _, c := range _commands {

		description, isHasArguments := c.Description()
		fmt.Printf("%s - %s\n", c.FlagSet().Name(), description)
		if isHasArguments {
			fmt.Printf("  ")
			c.FlagSet().Usage()
		}
		fmt.Println()
	}
}

func main() {
	logger.CanPrintToConsole(false)

	// initialize all possible commands
	addCommand(&cmdLogin{})
	addCommand(&cmdLogout{})
	addCommand(&cmdServers{})
	addCommand(&cmdFirewall{})
	addCommand(&cmdState{})
	addCommand(&cmdConnect{})
	addCommand(&cmdDisconnect{})

	if len(os.Args) < 2 {
		printUsageAll()
		os.Exit(1)
	}
	// initialize command handler
	port, secret, err := readDaemonPort()
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
	}

	_proto = protocol.CreateClient(port, secret)

	// process command
	isProcessed := false
	for _, c := range _commands {
		if c.FlagSet().Name() == os.Args[1] {
			isProcessed = true

			c.FlagSet().Parse(os.Args[2:])

			printHeader()

			fmt.Println(c.FlagSet().Name() + "...")
			//fmt.Printf("DEBUG: %+v\n", c)

			if err := c.Run(); err != nil {
				fmt.Printf("Error: %s\n", err.Error())
				if _, ok := err.(BadParameter); ok == true {
					c.FlagSet().Usage()
				}
				os.Exit(1)
			}

			fmt.Println("Success")
			break
		}
	}

	// unknown command
	if isProcessed == false {
		fmt.Printf("Error. Unexpected command %s\n", os.Args[1])
		printUsageAll()
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
			return 0, 0, fmt.Errorf("connection-info file not exists '%s'", file)
		}
		return 0, 0, fmt.Errorf("connection-info file existing check error '%s': %s", file, err)
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}

	vars := strings.Split(string(data), ":")
	if len(vars) != 2 {
		return 0, 0, fmt.Errorf("failed to parse connection-info file")
	}

	port, err = strconv.Atoi(strings.TrimSpace(vars[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse connection-info file: %w", err)
	}

	secret, err = strconv.ParseUint(strings.TrimSpace(vars[1]), 16, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse connection-info file: %w", err)
	}

	return port, secret, nil
}
