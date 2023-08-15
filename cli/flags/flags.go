//
//  IVPN command line interface (CLI)
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 IVPN Limited.
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

package flags

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
)

// NewFlagSetEx - create new command object
func NewFlagSetEx(name, description string) *CmdInfo {
	ret := &CmdInfo{}
	ret.Initialize(name, description)
	return ret
}

// CmdInfo contains info about single command with arguments
type CmdInfo struct {
	KeepArgsOrderInHelp bool // (when showing help) keep arguments in the same order as they were defined
	description         string
	fs                  *flag.FlagSet
	defaultArg          *string
	defaultArgName      string
	usageOrder          []string          // list of 'usage' text in the order flags have been defined
	argNames            map[string]string // variable name -> argument description (example: "-port PROTOCOL:PORT" => ["port"]"PROTOCOL:PORT")
	parseSpecial        func(arguments []string) bool
	preParse            func(arguments []string) (argumentsUpdated []string, err error)
	argIsAllowed        map[string]func() bool // variable name -> func which returns 'false' when argument is not applicable for current environment
}

// Initialize initialises object
func (c *CmdInfo) Initialize(name, description string) {
	c.argNames = make(map[string]string)

	c.description = description
	c.fs = flag.NewFlagSet(name, flag.ExitOnError)
	c.fs.Usage = func() {
		c.Usage(false)
	}
	c.argIsAllowed = make(map[string]func() bool)
}

// Parse parses flag definitions from the argument list
// see description of Flagset.Parse()
func (c *CmdInfo) Parse(arguments []string) error {
	if err := c.fs.Parse(arguments); err != nil {
		return err
	}

	if c.defaultArg != nil {
		// Looking for default argument
		// Only one argument allowed.
		if len(c.fs.Args()) > 1 {
			return BadParameter{}
		}
		if len(c.fs.Args()) > 0 {
			*c.defaultArg = c.fs.Args()[0]
		}
	} else if len(c.fs.Args()) > 0 {
		return BadParameter{}
	}
	return nil
}

func (c *CmdInfo) ParseSpecial(arguments []string) (haveParseSpecial bool) {
	if c.parseSpecial != nil {
		return c.parseSpecial(arguments)
	}
	return false
}

func (c *CmdInfo) SetParseSpecialFunc(f func(arguments []string) bool) {
	c.parseSpecial = f
}

func (c *CmdInfo) PreParse(arguments []string) (argumentsUpdated []string, err error) {
	if c.preParse != nil {
		return c.preParse(arguments)
	}
	return arguments, nil
}

func (c *CmdInfo) SetPreParseFunc(f func(arguments []string) (argumentsUpdated []string, err error)) {
	c.preParse = f
}

// NFlag returns the number of flags that have been set.
func (c *CmdInfo) NFlag() int { return c.fs.NFlag() }

// Name - command name
func (c *CmdInfo) Name() string { return c.fs.Name() }

// Description - command name
func (c *CmdInfo) Description() string { return c.description }

// Usage - prints command usage
func (c *CmdInfo) Usage(short bool) {
	fmt.Printf("Command usage:\n")
	c.usage(nil, short)
}

// UsageFormetted - prints command usage into tabwriter
func (c *CmdInfo) UsageFormetted(w *tabwriter.Writer, short bool) {
	c.usage(w, short)
}

func (c *CmdInfo) usage(w *tabwriter.Writer, short bool) {

	type flagInfo struct {
		DetailedName string
		Arg          string
	}

	tmpmap := make(map[string]flagInfo)

	keys := c.usageOrder
	// collect output date
	flagIterator := func(f *flag.Flag) {
		if flags, ok := tmpmap[f.Usage]; !ok {
			tmpmap[f.Usage] = flagInfo{DetailedName: "-" + f.Name, Arg: c.argNames[f.Name]}
		} else {
			flags.DetailedName = flags.DetailedName + "|-" + f.Name
			tmpmap[f.Usage] = flags
		}
	}
	c.fs.VisitAll(flagIterator)

	writer := w
	// create local writer (if not defined)
	if writer == nil {
		writer = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	}

	// Format output
	// command
	lines := strings.Split(c.Description(), "\n")
	fmt.Fprintln(writer, fmt.Sprintf("%s %s\t%s", c.Name(), c.defaultArgName, lines[0]))

	if short {
		return // Short help printed
	}

	// sorting keys (map is not sorted)
	if !c.KeepArgsOrderInHelp {
		sort.Strings(keys)
	}

	if len(lines) > 1 {
		for i := 1; i < len(lines); i++ {
			fmt.Fprintln(writer, fmt.Sprintf("  %s %s\t%s", "", "", lines[i]))
		}
	}

	// loop trough flags map
	for _, usage := range keys {
		flag := tmpmap[usage]

		// check if we can print info about this argument
		argNormalized := strings.Trim(flag.DetailedName, "- \n\r\t")
		if isArgAllowedFunc, ok := c.argIsAllowed[argNormalized]; ok && isArgAllowedFunc != nil && !isArgAllowedFunc() {
			// The argument is not applicable for current platform
			// Do not show it in usage
			continue
		}

		lines := strings.Split(usage, "\n")
		fmt.Fprintln(writer, fmt.Sprintf("  %s %s\t- %s", flag.DetailedName, flag.Arg, lines[0]))
		if len(lines) > 1 {
			for i := 1; i < len(lines); i++ {
				fmt.Fprintln(writer, fmt.Sprintf("  %s %s\t%s", "", "", lines[i]))
			}
		}
	}

	// in case of just created writer - flush it now
	if w == nil {
		writer.Flush()
	}
}

// DefaultStringVar defines default string argument
func (c *CmdInfo) DefaultStringVar(p *string, usage string) {
	c.defaultArgName = usage
	c.defaultArg = p
}

func (c *CmdInfo) saveFlagHelpOrder(usage string) {
	for _, v := range c.usageOrder {
		if v == usage {
			return
		}
	}
	c.usageOrder = append(c.usageOrder, usage)
}

// StringVar defines a string flag with specified name, default value, and usage string.
// The argument p points to a string variable in which to store the value of the flag.
func (c *CmdInfo) StringVar(p *string, name string, defValue string, argNAme string, usage string) {
	c.fs.StringVar(p, name, defValue, usage)
	c.argNames[name] = argNAme
	c.saveFlagHelpOrder(usage)
}
func (c *CmdInfo) StringVarEx(p *string, name string, defValue string, argNAme string, usage string, isAllowedFunc func() bool) {
	c.StringVar(p, name, defValue, argNAme, usage)
	if isAllowedFunc != nil {
		c.argIsAllowed[name] = isAllowedFunc
	}
}

// IntVar defines an int flag with specified name, default value, and usage string.
// The argument p points to an int variable in which to store the value of the flag.
func (c *CmdInfo) IntVar(p *int, name string, defValue int, argNAme string, usage string) {
	c.fs.IntVar(p, name, defValue, usage)
	c.argNames[name] = argNAme
	c.saveFlagHelpOrder(usage)
}
func (c *CmdInfo) IntVarEx(p *int, name string, defValue int, argNAme string, usage string, isAllowedFunc func() bool) {
	c.IntVar(p, name, defValue, argNAme, usage)
	if isAllowedFunc != nil {
		c.argIsAllowed[name] = isAllowedFunc
	}
}

// BoolVar defines a bool flag with specified name, default value, and usage string.
// The argument p points to a bool variable in which to store the value of the flag.
func (c *CmdInfo) BoolVar(p *bool, name string, defValue bool, usage string) {
	c.fs.BoolVar(p, name, defValue, usage)
	c.saveFlagHelpOrder(usage)
}
func (c *CmdInfo) BoolVarEx(p *bool, name string, defValue bool, usage string, isAllowedFunc func() bool) {
	c.BoolVar(p, name, defValue, usage)
	if isAllowedFunc != nil {
		c.argIsAllowed[name] = isAllowedFunc
	}
}
