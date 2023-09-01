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

//go:build debug
// +build debug

package helpers

import (
	"fmt"
	"runtime"
)

func GoroutinesCount() int {
	stackRecords := make([]runtime.StackRecord, 100)
	count, _ := runtime.GoroutineProfile(stackRecords)
	return count
}

// PrintGoroutinesInfo - prints to the console count of active goroutines and their stak traces
func PrintGoroutinesInfo() {
	stackRecords := make([]runtime.StackRecord, 100)
	count, ok := runtime.GoroutineProfile(stackRecords)
	if !ok {
		return
	}

	fmt.Println("----------------------")
	fmt.Println(fmt.Sprintf("Goroutines COUNT: %d", runtime.NumGoroutine()))
	for i := 0; i < count; i++ {
		fmt.Println("*** #", i+1)
		frames := stackRecords[i].Stack()

		for x := range frames {
			f := runtime.FuncForPC(frames[x])
			file, line := f.FileLine(frames[x])
			fmt.Println("\t", f.Name(), file, line)
		}
	}
	fmt.Println("----------------------")
}
