package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/ivpn/desktop-app-cli/commands"
	"github.com/ivpn/desktop-app-cli/flags"
	"github.com/ivpn/desktop-app-cli/protocol"
	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/service/platform"
	"github.com/ivpn/desktop-app-daemon/version"
)

// ICommand interface for command line command
type ICommand interface {
	Init()
	Parse(arguments []string) error
	Run() error

	Name() string
	Description() string
	Usage()
	UsageFormetted(w *tabwriter.Writer)
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
	fmt.Println("version:" + version.GetFullVersion())
}

func printUsageAll() {
	printHeader()
	fmt.Printf("Usage: %s COMMAND [OPTIONS...] [COMMAND_PARAMETER] [-h|-help]\n", filepath.Base(os.Args[0]))

	fmt.Println("COMANDS:")
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	for _, c := range _commands {
		c.UsageFormetted(writer)
	}
	writer.Flush()
}

func main() {
	logger.CanPrintToConsole(false)

	// initialize all possible commands
	stateCmd := commands.CmdState{}
	addCommand(&stateCmd)
	addCommand(&commands.CmdConnect{})
	addCommand(&commands.CmdDisconnect{})
	addCommand(&commands.CmdAccount{})
	addCommand(&commands.CmdServers{})
	addCommand(&commands.CmdFirewall{})

	// initialize command handler
	port, secret, err := readDaemonPort()
	if err != nil {
		fmt.Printf("ERROR: Unable to connect to service: %s\n", err)
		printServStartInstructions()
		os.Exit(1)
	}

	proto := protocol.CreateClient(port, secret)
	if err := proto.Connect(); err != nil {
		fmt.Printf("ERROR: Failed to connect to service : %s\n", err)
		printServStartInstructions()
		os.Exit(1)
	}

	commands.Initialize(proto)

	if len(os.Args) < 2 {
		if err := stateCmd.Run(); err != nil {
			fmt.Printf("Error: %v\n", err)
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
		if os.Args[1] != "-h" && os.Args[1] != "--h" && os.Args[1] != "-help" && os.Args[1] != "--help" {
			fmt.Printf("Error. Unexpected command %s\n", os.Args[1])
		}
		printUsageAll()
		os.Exit(1)
	}
}

func runCommand(c ICommand, args []string) {
	if err := c.Parse(args); err != nil {
		fmt.Printf("Error: %v\n", err)
		if _, ok := err.(flags.BadParameter); ok == true {
			c.Usage()
		}
		os.Exit(1)
	}

	if err := c.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		if _, ok := err.(flags.BadParameter); ok == true {
			c.Usage()
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
