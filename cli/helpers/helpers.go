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
	"strings"

	"github.com/ivpn/desktop-app/cli/flags"
)

func CheckIsAdmin() bool {
	return doCheckIsAdmin()
}

func BoolParameterParse(v string) (bool, error) {
	//if num, err := strconv.Atoi(v); err == nil && num > 0 {
	//	return true, nil
	//}
	val, _, err := BoolParameterParseEx(v, []string{"on", "true", "1"}, []string{"off", "false", "0"}, []string{})
	return val, err
}

func BoolParameterParseEx(v string, trueValues []string, falseValues []string, nullValue []string) (val bool, isNull bool, err error) {
	if len(trueValues) == 0 && len(falseValues) == 0 {
		return false, false, fmt.Errorf("internal error (bad arguments for BoolParameterParseEx)")
	}

	v = strings.ToLower(strings.TrimSpace(v))

	for _, tV := range trueValues {
		if v == strings.ToLower(strings.TrimSpace(tV)) {
			return true, false, nil
		}
	}

	for _, fV := range falseValues {
		if v == strings.ToLower(strings.TrimSpace(fV)) {
			return false, false, nil
		}
	}

	for _, nV := range nullValue {
		if v == strings.ToLower(strings.TrimSpace(nV)) {
			return false, true, nil
		}
	}

	// error: unsupported value
	infoSupportedNullVals := strings.Join(nullValue, "/")
	if len(infoSupportedNullVals) > 0 {
		infoSupportedNullVals = ", " + infoSupportedNullVals
	}
	return false, false, flags.BadParameter{Message: fmt.Sprintf("unsupported value '%s' for parameter (acceptable values: %s, %s%s)", v, strings.Join(falseValues, "/"), strings.Join(trueValues, "/"), infoSupportedNullVals)}
}

func TrimSpacesAndRemoveQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 1 {
		openSym := s[0]
		switch openSym {
		case '"', '\'', '`':
			if s[len(s)-1] == openSym {
				s = s[1 : len(s)-1]
			}
		default:
		}
	}
	return s
}
