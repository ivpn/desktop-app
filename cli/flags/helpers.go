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

import "strings"

// RemoveArgIfNoValue removes argument from the command line arguments if it has no value following it, otherwise,
// it returns the same arguments.
//
// This function is useful when dealing with string flags that are optional. It ensures that the argument flag is
// followed by a value. If not, it removes the flag from the list to avoid a parsing error when the flag parsing
// function expects a value.
//
// arguments: the original command line arguments
// argName: the name of the argument to check for a following value
//
// Returns:
// - newArgs: a new slice of arguments excluding any instances of argName not followed by a value
// - isRemoved: a boolean indicating whether an argument was removed
//
// EXAMPLE:
//
//		c.SetPreParseFunc(c.preParse)
//
//		func (c *CmdAntitracker) preParse(arguments []string) ([]string, error) {
//			isArgRemoved := false
//			arguments, isArgRemoved = flags.RemoveArgIfNoValue(arguments, "-on")
//			if isArgRemoved {
//				c.on = EmptyBlockListName
//			}
//			arguments, isArgRemoved = flags.RemoveArgIfNoValue(arguments, "-on_hardcore")
//			if isArgRemoved {
//				c.hardcore = EmptyBlockListName
//			}
//			return arguments, nil
//	}
func RemoveArgIfNoValue(arguments []string, argName string) (newArgs []string, isRemoved bool) {
	argName = strings.TrimLeft(argName, "-")
	shortArgName := "-" + argName
	longArgName := "--" + argName

	lastIdx := len(arguments) - 1
	emptyArgIdx := -1
	for idx, a := range arguments {
		if a == shortArgName || a == longArgName {
			if idx == lastIdx {
				emptyArgIdx = idx // empty argument is detected
				break
			}

			// Check if argument is alone:
			// - next argument after '-o' must start with '-' (must be new arg definition)
			nextIdx := idx + 1

			if nextIdx <= lastIdx {
				nextArg := arguments[nextIdx]
				if !strings.HasPrefix(nextArg, "-") { // next argument after must start with '-'
					break
				}
				emptyArgIdx = idx // empty argument is detected
			}
			break
		}
	}

	if emptyArgIdx >= 0 {
		// If argument requires data, but it is empty, the argument must be removed in order to avoid parsing error
		return append(arguments[:emptyArgIdx], arguments[emptyArgIdx+1:]...), true
	}

	return arguments, false
}
