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

package winlib_test

import (
	"fmt"
	"syscall"
	"testing"

	"github.com/ivpn/desktop-app/daemon/service/firewall/winlib"
)

func TestFuncsCall(t *testing.T) {
	DoTest()
}

func DoTest() {
	defer func() {
		if r := recover(); r != nil {

		}
	}()

	sess, errS1 := winlib.CreateWfpSessionObject(true)
	fmt.Println(sess, errS1)

	engine, errEn1 := winlib.WfpEngineOpen(sess)
	fmt.Println(engine, errEn1)

	guid := syscall.GUID{
		Data1: 0xfed0afd4,
		Data2: 0x98d4,
		Data3: 0x4233,
		Data4: [8]byte{0xa4, 0xf3, 0x8b, 0x7c, 0x02, 0x44, 0x50, 0x01},
	}

	isInstalled, flags, err := winlib.WfpGetProviderFlags(engine, guid)
	fmt.Println(isInstalled, flags, err)

	sublayer, serr := winlib.FWPMSUBLAYER0Create(guid, 0)
	fmt.Println(sublayer, serr)

	isInstalled, err = winlib.WfpSubLayerIsInstalled(engine, guid)
	fmt.Println(isInstalled, err)

	winlib.FWPMSUBLAYER0SetDisplayData(sublayer, "sbName", "sbDescription")
	fmt.Println(err)

	err = winlib.WfpSubLayerAdd(engine, sublayer)
	fmt.Println(err)

	isInstalled, err = winlib.WfpSubLayerIsInstalled(engine, guid)
	fmt.Println(isInstalled, err)

	winlib.WfpSubLayerDelete(engine, guid)
	fmt.Println(err)

	provider, perr := winlib.FWPMPROVIDER0Create(guid)
	fmt.Println(provider, perr)

	err = winlib.FWPMPROVIDER0SetFlags(provider, 1)
	fmt.Println(err)

	err = winlib.FWPMPROVIDER0SetDisplayData(provider, "theName", "theDESC")
	fmt.Println(err)

	err = winlib.WfpProviderAdd(engine, provider)
	fmt.Println(err)

	err = winlib.WfpProviderDelete(engine, guid)
	fmt.Println(err)

	isInstalled, flags, err = winlib.WfpGetProviderFlags(engine, guid)
	fmt.Println(isInstalled, flags, err)

	err = winlib.WfpTransactionCommit(engine)
	fmt.Println(err)

	err = winlib.WfpTransactionAbort(engine)
	fmt.Println(err)

	err = winlib.WfpTransactionBegin(engine)
	fmt.Println(err)
	err = winlib.WfpTransactionCommit(engine)
	fmt.Println(err)

	err = winlib.WfpTransactionBegin(engine)
	fmt.Println(err)
	err = winlib.WfpTransactionAbort(engine)
	fmt.Println(err)

	errEC := winlib.WfpEngineClose(engine)
	fmt.Println(errEC)

	winlib.DeleteWfpSessionObject(sess)
	return
}
