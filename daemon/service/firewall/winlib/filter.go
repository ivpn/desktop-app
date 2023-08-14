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

//go:build windows
// +build windows

package winlib

import (
	"syscall"
)

// Condition represents filter condition
type Condition interface {
	Apply(filter syscall.Handle, conditionIndex uint32) error
}

// Filter - WFP filter
type Filter struct {

	// TODO: make fiels not visible outside of package

	Key         syscall.GUID
	KeyLayer    syscall.GUID
	KeySublayer syscall.GUID
	KeyProvider syscall.GUID

	DisplayDataName        string
	DisplayDataDescription string

	Action     FwpActionType
	Weight     byte
	Flags      FwpmFilterFlags
	Conditions []Condition
}

// NewFilter - create new filter
func NewFilter(
	keyProvider syscall.GUID,
	keyLayer syscall.GUID,
	keySublayer syscall.GUID,
	dispName string,
	dispDescription string) Filter {

	return Filter{
		Key:                    NewGUID(),
		Conditions:             make([]Condition, 0, 1),
		KeyProvider:            keyProvider,
		KeyLayer:               keyLayer,
		KeySublayer:            keySublayer,
		DisplayDataName:        dispName,
		DisplayDataDescription: dispDescription}
}

// AddCondition adds filter condition
func (f *Filter) AddCondition(c Condition) {
	f.Conditions = append(f.Conditions, c)
}

// SetDisplayData adds filter display data
func (f *Filter) SetDisplayData(name string, description string) {
	f.DisplayDataName = name
	f.DisplayDataDescription = description
}
