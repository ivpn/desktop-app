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
	"github.com/stretchr/testify/assert"
)

// Use assert.ElementsMatch for comparing slices, but with a bool result.
type dummyt struct{}

func (t dummyt) Errorf(string, ...interface{}) {}

func SliceElementsMatch(listA, listB interface{}) bool {
	if listA == nil && listB == nil {
		return true
	}
	if listA == nil || listB == nil {
		return false
	}
	return assert.ElementsMatch(dummyt{}, listA, listB)
}
