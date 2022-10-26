//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
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
	return BoolParameterParseEx(v, []string{"on", "true", "1"}, []string{"off, false, 0"})
}

func BoolParameterParseEx(v string, trueValues []string, falseValues []string) (bool, error) {
	if len(trueValues) == 0 && len(falseValues) == 0 {
		return false, fmt.Errorf("internal error (bad arguments for BoolParameterParseEx)")
	}

	v = strings.ToLower(strings.TrimSpace(v))

	for _, tV := range trueValues {
		if v == strings.ToLower(strings.TrimSpace(tV)) {
			return true, nil
		}
	}

	for _, fV := range falseValues {
		if v == strings.ToLower(strings.TrimSpace(fV)) {
			return false, nil
		}
	}

	return false, flags.BadParameter{Message: fmt.Sprintf("unsupported value '%s' for boolean parameter (acceptable values: %s, %s)", v, strings.Join(falseValues, "/"), strings.Join(trueValues, "/"))}
}

func BoolToStr(v *bool, trueVal, falseVal, nullVal string) string {
	if v == nil {
		return nullVal
	}

	if *v {
		return trueVal
	}
	return falseVal
}
