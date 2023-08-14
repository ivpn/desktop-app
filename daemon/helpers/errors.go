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

package helpers

import (
	"fmt"
	"runtime"
)

// ErrorNotImplemented - functionality not implemented
type ErrorNotImplemented struct {
	Caller string
}

// NewErrNotImplemented - create not implemented error
func NewErrNotImplemented() ErrorNotImplemented {
	var caller string
	var err error
	if caller, err = getCallerMethodName(); err != nil {
		caller = ""
	}
	return ErrorNotImplemented{Caller: caller}
}

func (e ErrorNotImplemented) Error() string {
	if len(e.Caller) > 0 {
		return fmt.Sprintf("not implemented (%s)", e.Caller)
	}
	return "not implemented (%s)"
}

func getCallerMethodName() (string, error) {
	fpcs := make([]uintptr, 1)
	// Skip 3 levels to get the caller
	n := runtime.Callers(3, fpcs)
	if n == 0 {
		return "", fmt.Errorf("no caller")
	}

	caller := runtime.FuncForPC(fpcs[0] - 1)
	if caller == nil {
		return "", fmt.Errorf("msg caller is nil")
	}

	return caller.Name(), nil
}
