//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2021 Privatus Limited.
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

package splittun

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/ivpn/desktop-app/daemon/service/platform"
)

var (
	// error describing details if functionality not available
	funcNotAvailableError            error
	fSplitTun_Connect                *syscall.LazyProc
	fSplitTun_Disconnect             *syscall.LazyProc
	fSplitTun_StopAndClean           *syscall.LazyProc
	fSplitTun_SplitStart             *syscall.LazyProc
	fSplitTun_GetState               *syscall.LazyProc
	fSplitTun_ConfigSetAddresses     *syscall.LazyProc
	fSplitTun_ConfigGetAddresses     *syscall.LazyProc
	fSplitTun_ConfigSetSplitAppRaw   *syscall.LazyProc
	fSplitTun_ConfigGetSplitAppRaw   *syscall.LazyProc
	fSplitTun_ProcMonInitRunningApps *syscall.LazyProc
	//fSplitTun_ProcMonStart           *syscall.LazyProc
	//fSplitTun_ProcMonStop            *syscall.LazyProc
	//fSplitTun_SplitStop              *syscall.LazyProc
)

var (
	isDriverConnected bool
)

type ConfigApps struct {
	ImagesPathToSplit []string
}

type Config struct {
	Addr ConfigAddresses
	Apps ConfigApps
}

// Initialize doing initialization stuff (called on application start)
func implInitialize() error {
	wfpDllPath := platform.WindowsWFPDllPath()
	if len(wfpDllPath) == 0 {
		return fmt.Errorf("unable to initialize split-tunnelling wrapper: firewall dll path not initialized")
	}
	if _, err := os.Stat(wfpDllPath); err != nil {
		return fmt.Errorf("unable to initialize split-tunnelling wrapper (firewall dll not found) : '%s'", wfpDllPath)
	}

	dll := syscall.NewLazyDLL(wfpDllPath)

	fSplitTun_Connect = dll.NewProc("SplitTun_Connect")
	fSplitTun_Disconnect = dll.NewProc("SplitTun_Disconnect")
	fSplitTun_StopAndClean = dll.NewProc("SplitTun_StopAndClean")
	fSplitTun_ProcMonInitRunningApps = dll.NewProc("SplitTun_ProcMonInitRunningApps")
	fSplitTun_SplitStart = dll.NewProc("SplitTun_SplitStart")
	fSplitTun_GetState = dll.NewProc("SplitTun_GetState")
	fSplitTun_ConfigSetAddresses = dll.NewProc("SplitTun_ConfigSetAddresses")
	fSplitTun_ConfigGetAddresses = dll.NewProc("SplitTun_ConfigGetAddresses")
	fSplitTun_ConfigSetSplitAppRaw = dll.NewProc("fSplitTun_ConfigSetSplitAppRaw")
	fSplitTun_ConfigGetSplitAppRaw = dll.NewProc("fSplitTun_ConfigGetSplitAppRaw")
	//fSplitTun_ProcMonStart = dll.NewProc("SplitTun_ProcMonStart")
	//fSplitTun_ProcMonStop = dll.NewProc("SplitTun_ProcMonStop")
	//fSplitTun_SplitStop = dll.NewProc("SplitTun_SplitStop")

	// to ensure that functionality works - just try to start/stop driver
	defer disconnect(false)
	if connectErr := connect(false); connectErr != nil {
		funcNotAvailableError = fmt.Errorf("Split-Tunnel functionality test failed: %w", connectErr)
	}

	return funcNotAvailableError
}

func implFuncNotAvailableError() (generalStError, inversedStError error) {
	return funcNotAvailableError, fmt.Errorf("Inversed Split-Tunnelling is not implemented for this platform")
}

func implReset() error {
	// not applicable for Windows. Same effect has implApplyConfig(false, , , [])
	return nil
}

func implApplyConfig(isStEnabled bool, isStInversed bool, isVpnEnabled bool, addrConfig ConfigAddresses, splitTunnelApps []string) error {
	splitTunErr, splitTunInversedErr := GetFuncNotAvailableError()
	if splitTunErr != nil || (isStInversed && splitTunInversedErr != nil) {
		// Split-Tunneling not accessable (not able to connect to a driver or not implemented for current platform)
		return nil
	}

	// If ST connected:
	//	- stop and erase old configuration
	//  - if ST have to be disabled (or VPN is not connected) - disconnect ST driver
	if isDriverConnected {
		if err := stopAndClean(); err != nil {
			return log.ErrorE(fmt.Errorf("failed to clean split-tunnelling state: %w", err), 0)
		}
		if !isStEnabled || !isVpnEnabled {
			if err := disconnect(true); err != nil {
				return log.ErrorE(fmt.Errorf("failed to clean split-tunnelling state: %w", err), 0)
			}
		}
	}

	if !isVpnEnabled {
		// VPN not connected. No sense to enable split-tunnelling
		return nil
	}

	if !isStEnabled || len(splitTunnelApps) == 0 || ((addrConfig.IPv4Public == nil || addrConfig.IPv4Tunnel == nil) && (addrConfig.IPv6Public == nil || addrConfig.IPv6Tunnel == nil)) {
		// no configuration
		return nil
	}

	// If ST not connected:
	//	- connect driver
	//  - stop and erase old configuration
	if !isDriverConnected {
		if err := connect(true); err != nil {
			return log.ErrorE(fmt.Errorf("failed to start split-tunnelling: %w", err), 0)
		}
		if err := stopAndClean(); err != nil {
			return log.ErrorE(fmt.Errorf("failed to clean split-tunnelling state: %w", err), 0)
		}
	}

	// For inversed split-tunnel we just inverse IP addresses in driver configuration (defaultPublicInterfaceIP <=> tunnelInterfaceIP)
	if isStInversed {
		p4 := addrConfig.IPv4Public
		addrConfig.IPv4Public = addrConfig.IPv4Tunnel
		addrConfig.IPv4Tunnel = p4

		p6 := addrConfig.IPv6Public
		addrConfig.IPv6Public = addrConfig.IPv6Tunnel
		addrConfig.IPv6Tunnel = p6
	}

	// Set new configuration
	cfg := Config{}
	cfg.Apps = ConfigApps{ImagesPathToSplit: splitTunnelApps}
	cfg.Addr = addrConfig

	if err := setConfig(cfg); err != nil {
		log.Error(fmt.Errorf("error on configuring Split-Tunnelling: %w", err))
	} else {
		if err := start(); err != nil {
			log.Error(fmt.Errorf("error on start Split-Tunnelling: %w", err))
		} else {
			log.Info(fmt.Sprintf("Split-Tunnelling started: IPv4: (%s) => (%s) IPv6: (%s) => (%s)", addrConfig.IPv4Tunnel, addrConfig.IPv4Public, addrConfig.IPv6Tunnel, addrConfig.IPv6Public))
		}
	}

	return nil
}

func implAddPid(pid int, commandToExecute string) error {
	return fmt.Errorf("operation not applicable for current platform")
}

func implRemovePid(pid int) error {
	return fmt.Errorf("operation not applicable for current platform")
}

func implGetRunningApps() ([]RunningApp, error) {
	return nil, fmt.Errorf("operation not applicable for current platform")
}

func catchPanic(err *error) {
	if r := recover(); r != nil {
		log.Error("PANIC (recovered): ", r)
		if e, ok := r.(error); ok {
			*err = e
		} else {
			*err = errors.New(fmt.Sprint(r))
		}
	}
}

func checkCallErrResp(retval uintptr, err error, mName string) error {
	if err != syscall.Errno(0) {
		return log.ErrorE(fmt.Errorf("%s:  %w", mName, err), 1)
	}
	if retval != 1 {
		return log.ErrorE(fmt.Errorf("Split-Tunnelling operation failed (%s)", mName), 1)
	}
	return nil
}

func connect(logging bool) (err error) {
	defer catchPanic(&err)

	if isDriverConnected {
		return nil
	}

	if logging {
		log.Info("Split-Tunnelling: Connect driver...")
	}

	drvPath := platform.WindowsSplitTunnelDriverPath()
	utfDrvPath, err := syscall.UTF16PtrFromString(drvPath)
	if err != nil {
		return fmt.Errorf("(SplitTun_Connect) Failed to convert driver file path: %w", err)
	}

	retval, _, err := fSplitTun_Connect.Call(uintptr(unsafe.Pointer(utfDrvPath)))
	if err != syscall.Errno(0) {
		if err == syscall.ERROR_FILE_NOT_FOUND {
			err = fmt.Errorf("%w (check if IVPN Split-Tunnelling driver installed)", err)
		}
		return err
	}
	if retval != 1 {
		return fmt.Errorf("Split-Tunnelling operation failed (SplitTun_Connect)")
	}

	isDriverConnected = true
	if logging {
		log.Info("Split-Tunnelling: driver ready")
	}

	return nil
}

func disconnect(logging bool) (err error) {
	defer catchPanic(&err)

	if logging {
		log.Info("Split-Tunnelling: Disconnect driver...")
	}
	isDriverConnected = false

	retval, _, err := fSplitTun_Disconnect.Call()
	if err := checkCallErrResp(retval, err, "SplitTun_Disconnect"); err != nil {
		return err
	}

	return nil
}

func stopAndClean() (err error) {
	defer catchPanic(&err)

	log.Info("Split-Tunnelling: StopAndClean...")

	/// Stop and clean everything:
	///		Stop splitting
	///		Stop processes monitoring
	///		Clean all configuration/statuses
	retval, _, err := fSplitTun_StopAndClean.Call()
	if err := checkCallErrResp(retval, err, "SplitTun_StopAndClean"); err != nil {
		return err
	}

	return nil
}

func start() (err error) {
	defer catchPanic(&err)

	log.Info("Split-Tunnelling: Start...")

	/// Start splitting.
	/// If "process monitor" not running - it will be started.
	///
	/// Operation fails when configuration not complete:
	///		- no splitting apps defined
	///		- no IP configuration (IP-public + IP-tunnel) defined at least for one protocol type (IPv4\IPv6)
	///
	/// If only IPv4 configuration defined - splitting will work only for IPv4
	/// If only IPv6 configuration defined - splitting will work only for IPv6
	retval, _, err := fSplitTun_SplitStart.Call()
	if err := checkCallErrResp(retval, err, "SplitTun_SplitStart"); err != nil {
		return err
	}

	// Initialize already running apps info
	/// Set application PID\PPIDs which have to be splitted.
	/// It adds new info to internal process tree but not erasing current known PID\PPIDs.
	/// Operaion fails when 'process monitor' not running
	retval, _, err = fSplitTun_ProcMonInitRunningApps.Call()

	if err == syscall.ERROR_NO_MORE_FILES {
		// ignore ERROR_NO_MORE_FILES error. It is Ok after calling of 'SplitTun_ProcMonInitRunningApps'
		err = syscall.Errno(0)
	}
	if err := checkCallErrResp(retval, err, "SplitTun_ProcMonInitRunningApps"); err != nil {
		return err
	}

	return nil
}

func setConfig(config Config) (err error) {
	defer catchPanic(&err)

	log.Info("Split-Tunnelling: SetConfig...")

	// SET IP ADDRESSES
	IPv4Public := config.Addr.IPv4Public.To4()
	IPv4Tunnel := config.Addr.IPv4Tunnel.To4()
	IPv6Public := config.Addr.IPv6Public.To16()
	IPv6Tunnel := config.Addr.IPv6Tunnel.To16()

	if IPv4Public == nil {
		IPv4Public = make([]byte, 4)
	}
	if IPv4Tunnel == nil {
		IPv4Tunnel = make([]byte, 4)
	}
	if IPv6Public == nil {
		IPv6Public = make([]byte, 16)
	}
	if IPv6Tunnel == nil {
		IPv6Tunnel = make([]byte, 16)
	}

	retval, _, err := fSplitTun_ConfigSetAddresses.Call(
		uintptr(unsafe.Pointer(&IPv4Public[0])),
		uintptr(unsafe.Pointer(&IPv4Tunnel[0])),
		uintptr(unsafe.Pointer(&IPv6Public[0])),
		uintptr(unsafe.Pointer(&IPv6Tunnel[0])))
	if err := checkCallErrResp(retval, err, "SplitTun_ConfigSetAddresses"); err != nil {
		return err
	}

	// SET APPS TO SPLIT
	buff, err := makeRawBuffAppsConfig(config.Apps)
	if err != nil {
		return log.ErrorE(fmt.Errorf("failed to set split-tinnelling configuration (apps): %w", err), 0)
	}

	var bufSize uint32 = uint32(len(buff))
	retval, _, err = fSplitTun_ConfigSetSplitAppRaw.Call(
		uintptr(unsafe.Pointer(&buff[0])),
		uintptr(bufSize))
	if err := checkCallErrResp(retval, err, "SplitTun_ConfigSetSplitAppRaw"); err != nil {
		return err
	}

	return nil
}

func makeRawBuffAppsConfig(apps ConfigApps) (bytesArr []byte, err error) {
	//	DWORD common size bytes
	//	DWORD strings cnt
	//	DWORD str1Len
	//	DWORD str2Len
	//	...
	//	WCHAR[] str1
	//	WCHAR[] str2
	//	...

	sizesBuff := new(bytes.Buffer)
	stringsBuff := new(bytes.Buffer)

	var strLen uint32 = 0
	for _, path := range apps.ImagesPathToSplit {
		uint16arr, _ := syscall.UTF16FromString(path)
		// remove NULL-termination
		uint16arr = uint16arr[:len(uint16arr)-1]

		strLen = uint32(len(uint16arr))
		if err := binary.Write(sizesBuff, binary.LittleEndian, strLen); err != nil {
			return nil, err
		}
		if err := binary.Write(stringsBuff, binary.LittleEndian, uint16arr); err != nil {
			return nil, err
		}
	}

	var totalSize uint32 = uint32(4 + 4 + sizesBuff.Len() + stringsBuff.Len())
	var stringsCnt uint32 = uint32(len(apps.ImagesPathToSplit))

	buff := new(bytes.Buffer)
	if err := binary.Write(buff, binary.LittleEndian, totalSize); err != nil {
		return nil, err
	}
	if err := binary.Write(buff, binary.LittleEndian, stringsCnt); err != nil {
		return nil, err
	}
	if err := binary.Write(buff, binary.LittleEndian, sizesBuff.Bytes()); err != nil {
		return nil, err
	}
	if err := binary.Write(buff, binary.LittleEndian, stringsBuff.Bytes()); err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

/*
func getConfig() (cfg Config, err error) {
	defer catchPanic(&err)

	// ADDRESSES
	IPv4Public := make([]byte, 4)
	IPv4Tunnel := make([]byte, 4)
	IPv6Public := make([]byte, 16)
	IPv6Tunnel := make([]byte, 16)

	retval, _, err := fSplitTun_ConfigGetAddresses.Call(
		uintptr(unsafe.Pointer(&IPv4Public[0])),
		uintptr(unsafe.Pointer(&IPv4Tunnel[0])),
		uintptr(unsafe.Pointer(&IPv6Public[0])),
		uintptr(unsafe.Pointer(&IPv6Tunnel[0])))
	if err := checkCallErrResp(retval, err, "SplitTun_ConfigGetAddresses"); err != nil {
		return Config{}, err
	}

	addr := ConfigAddresses{IPv4Public: IPv4Public, IPv4Tunnel: IPv4Tunnel, IPv6Public: IPv6Public, IPv6Tunnel: IPv6Tunnel}

	// APPS

	// get required buffer size
	var buffSize uint32 = 0
	var emptyBuff []byte
	_, _, err = fSplitTun_ConfigGetSplitAppRaw.Call(
		uintptr(unsafe.Pointer(&emptyBuff)),
		uintptr(unsafe.Pointer(&buffSize)))
	if err := checkCallErrResp(retval, err, "SplitTun_ConfigGetSplitAppRaw"); err != nil {
		return Config{}, err
	}

	// get data
	buff := make([]byte, buffSize)
	retval, _, err = fSplitTun_ConfigGetSplitAppRaw.Call(
		uintptr(unsafe.Pointer(&buff[0])),
		uintptr(unsafe.Pointer(&buffSize)))
	if err := checkCallErrResp(retval, err, "SplitTun_ConfigGetSplitAppRaw"); err != nil {
		return Config{}, err
	}

	apps, err := parseRawBuffAppsConfig(buff)
	if err != nil {
		return Config{}, log.ErrorE(fmt.Errorf("failed to obtain split-tinnelling configuration (apps): %w", err), 0)
	}

	return Config{Addr: addr, Apps: apps}, nil
}
func parseRawBuffAppsConfig(bytesArr []byte) (apps ConfigApps, err error) {
	//	DWORD common size bytes
	//	DWORD strings cnt
	//	DWORD str1Len
	//	DWORD str2Len
	//	...
	//	WCHAR[] str1
	//	WCHAR[] str2
	//	...

	var totalSize uint32
	var stringsCnt uint32
	files := make([]string, 0)

	buff := bytes.NewReader(bytesArr)
	if err := binary.Read(buff, binary.LittleEndian, &totalSize); err != nil {
		return ConfigApps{}, err
	}
	if err := binary.Read(buff, binary.LittleEndian, &stringsCnt); err != nil {
		return ConfigApps{}, err
	}

	if int(totalSize) != len(bytesArr) {
		return ConfigApps{}, fmt.Errorf("failed to parse split-tun configuration (applications)")
	}

	buffSizes := bytes.NewReader(bytesArr[4+4 : 4+4+stringsCnt*4])
	buffStrings := bytes.NewReader(bytesArr[4+4+stringsCnt*4:])

	var i uint32
	var strBytesSize uint32
	for i = 0; i < stringsCnt; i++ {
		if err := binary.Read(buffSizes, binary.LittleEndian, &strBytesSize); err != nil {
			return ConfigApps{}, err
		}

		uint16str := make([]uint16, strBytesSize)
		if err := binary.Read(buffStrings, binary.LittleEndian, &uint16str); err != nil {
			return ConfigApps{}, err
		}

		files = append(files, syscall.UTF16ToString(uint16str))
	}

	return ConfigApps{ImagesPathToSplit: files}, nil
}

func implGetState() (state State, err error) {
	defer catchPanic(&err)

	var isConfigOk uint32
	var isEnabledProcessMonitor uint32
	var isEnabledSplitting uint32

	retval, _, err := fSplitTun_GetState.Call(
		uintptr(unsafe.Pointer(&isConfigOk)),
		uintptr(unsafe.Pointer(&isEnabledProcessMonitor)),
		uintptr(unsafe.Pointer(&isEnabledSplitting)))
	if err := checkCallErrResp(retval, err, "fSplitTun_GetState"); err != nil {
		return State{}, err
	}

	return State{IsConfigOk: isConfigOk != 0, IsEnabledSplitting: isEnabledSplitting != 0}, nil
}

func implStop() (err error) {
	defer catchPanic(&err)

	// stop splitting
	retval, _, err := fSplitTun_SplitStop.Call()
	if err := checkCallErrResp(retval, err, "SplitTun_SplitStop"); err != nil {
		return err
	}

	// stop process monitor
	retval, _, err = fSplitTun_ProcMonStop.Call()
	if err := checkCallErrResp(retval, err, "SplitTun_ProcMonStop"); err != nil {
		return err
	}

	return nil
}*/
