//
//  IVPN command line interface (CLI)
//  https://github.com/ivpn/desktop-app-cli
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
	description    string
	fs             *flag.FlagSet
	defaultArg     *string
	defaultArgName string
	argNames       map[string]string // variable name -> argument description
}

// Initialize initialises object
func (c *CmdInfo) Initialize(name, description string) {
	c.argNames = make(map[string]string)
	c.description = description
	c.fs = flag.NewFlagSet(name, flag.ExitOnError)
	c.fs.Usage = func() {
		c.Usage(false)
	}
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
		ArgDesc      string
	}

	tmpmap := make(map[string]flagInfo)

	// collect output date
	flagIterator := func(f *flag.Flag) {
		if flags, ok := tmpmap[f.Usage]; ok == false {
			tmpmap[f.Usage] = flagInfo{DetailedName: "-" + f.Name, ArgDesc: c.argNames[f.Name]}
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

	// sorting keys (map is not sorted)
	keys := make([]string, 0, len(tmpmap))
	for k := range tmpmap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Format output
	// command
	lines := strings.Split(c.Description(), "\n")
	fmt.Fprintln(writer, fmt.Sprintf("%s %s\t%s", c.Name(), c.defaultArgName, lines[0]))
	if short {
		return
	}
	if len(lines) > 1 {
		for i := 1; i < len(lines); i++ {
			fmt.Fprintln(writer, fmt.Sprintf("  %s %s\t%s", "", "", lines[i]))
		}
	}

	// loop trough flags map
	for _, usage := range keys {
		flag, _ := tmpmap[usage]
		lines := strings.Split(usage, "\n")
		fmt.Fprintln(writer, fmt.Sprintf("  %s %s\t- %s", flag.DetailedName, flag.ArgDesc, lines[0]))
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

// StringVar defines a string flag with specified name, default value, and usage string.
// The argument p points to a string variable in which to store the value of the flag.
func (c *CmdInfo) StringVar(p *string, name string, defValue string, argNAme string, usage string) {
	c.fs.StringVar(p, name, defValue, usage)
	c.argNames[name] = argNAme
}

// IntVar defines an int flag with specified name, default value, and usage string.
// The argument p points to an int variable in which to store the value of the flag.
func (c *CmdInfo) IntVar(p *int, name string, defValue int, argNAme string, usage string) {
	c.fs.IntVar(p, name, defValue, usage)
	c.argNames[name] = argNAme
}

// BoolVar defines a bool flag with specified name, default value, and usage string.
// The argument p points to a bool variable in which to store the value of the flag.
func (c *CmdInfo) BoolVar(p *bool, name string, defValue bool, usage string) {
	c.fs.BoolVar(p, name, defValue, usage)
}
