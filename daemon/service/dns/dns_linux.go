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

package dns

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/ivpn/desktop-app/daemon/helpers"
	"github.com/ivpn/desktop-app/daemon/service/dns/dnscryptproxy"
	"github.com/ivpn/desktop-app/daemon/service/platform"
)

var (
	resolvFile             string      = "/etc/resolv.conf"
	resolvBackupFile       string      = "/etc/resolv.conf.ivpnsave"
	defaultFilePermissions os.FileMode = 0644

	isPaused  bool = false
	manualDNS DnsSettings

	dnscryptProxyProcess *dnscryptproxy.DnsCryptProxy

	done chan struct{}
)

func init() {
	done = make(chan struct{})
}

// implInitialize doing initialization stuff (called on application start)
func implInitialize() error {
	// check if backup DNS file exists
	if _, err := os.Stat(resolvBackupFile); err != nil {
		// nothing to restore
		return nil
	}

	log.Info("Detected DNS configuration from the previous VPN connection. Restoring OS-default DNS values ...")
	// restore it
	if err := implDeleteManual(nil); err != nil {
		return fmt.Errorf("failed to restore DNS to default: %w", err)
	}

	return nil
}

func implPause() error {
	if !isBackupExists(resolvBackupFile) {
		// The backup for the OS-defined configuration not exists.
		// It seems, we are not connected. Nothing to pause.
		return nil
	}

	// stop file change monitoring
	stopDNSChangeMonitoring()

	// restore original OS-default DNS configuration
	// (the backup file will not be deleted)
	isDeleteBackup := false // do not delete backup file
	ret := restoreBackup(resolvBackupFile, isDeleteBackup)

	isPaused = true
	return ret
}

func implResume(defaultDNS DnsSettings) error {
	isPaused = false

	if !manualDNS.IsEmpty() {
		// set manual DNS (if defined)
		return implSetManual(manualDNS, nil)
	}

	if !defaultDNS.IsEmpty() {
		return implSetManual(defaultDNS, nil)
	}

	return nil
}

func implGetDnsEncryptionAbilities() (dnsOverHttps, dnsOverTls bool, err error) {
	return true, false, nil
}

// Set manual DNS.
// 'localInterfaceIP' - not in use for Linux implementation
func implSetManual(dnsCfg DnsSettings, localInterfaceIP net.IP) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("PANIC (recovered): ", r)
			retErr = fmt.Errorf("%v", r)
			if err, ok := r.(error); ok {
				log.ErrorTrace(err)
			}
		}

		if retErr != nil {
			stopDnscryptProxyProcess()
		}
	}()
	if isPaused {
		// in case of PAUSED state -> just save manualDNS config
		// it will be applied on RESUME
		manualDNS = dnsCfg
		return nil
	}

	stopDNSChangeMonitoring()

	stopDnscryptProxyProcess()

	if dnsCfg.IsEmpty() {
		return implDeleteManual(nil)
	}

	if dnsCfg.Encryption != EncryptionNone {

		// Configure + start dnscrypt-proxy

		// Generate DNS server stamp
		var stamp dnscryptproxy.ServerStamp
		switch dnsCfg.Encryption {
		case EncryptionDnsOverHttps:
			stamp.Proto = dnscryptproxy.StampProtoTypeDoH
		default:
			return fmt.Errorf("unsupported DNS encryption type")
		}

		//stamp.Props |= dnscryptproxy.ServerInformalPropertyDNSSEC
		//stamp.Props |= dnscryptproxy.ServerInformalPropertyNoLog
		//stamp.Props |= dnscryptproxy.ServerInformalPropertyNoFilter

		stamp.ServerAddrStr = dnsCfg.DnsHost

		u, err := url.Parse(dnsCfg.DohTemplate)
		if err != nil {
			return err
		}

		if u.Scheme != "https" {
			return fmt.Errorf("bad template URL scheme: " + u.Scheme)
		}
		stamp.ProviderName = u.Host
		stamp.Path = u.Path

		binPath, configPathTemplate, configPathMutable := platform.DnsCryptProxyInfo()
		// generate dnscrypt-proxy configuration
		if err = dnscryptproxy.SaveConfigFile(stamp.String(), configPathTemplate, configPathMutable); err != nil {
			return err
		}

		dnscryptProxyProcess = dnscryptproxy.CreateDnsCryptProxy(binPath, configPathMutable)

		if err = dnscryptProxyProcess.Start(); err != nil {
			dnscryptProxyProcess.Stop()
			dnscryptProxyProcess = nil

			return fmt.Errorf("failed to start dnscrypt-proxy: %w", err)
		}

		// the local DNS must be configured to the dnscrypt-proxy (localhost)
		dnsCfg = DnsSettings{DnsHost: "127.0.0.1"}
	}

	createBackupIfNotExists := func() (created bool, er error) {
		isOwerwriteIfExists := false
		return createBackup(resolvBackupFile, isOwerwriteIfExists)
	}

	saveNewConfig := func() error {
		createBackupIfNotExists()

		// create new configuration
		out, err := os.OpenFile(resolvFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, defaultFilePermissions)
		if err != nil {
			return fmt.Errorf("failed to update DNS configuration (%w)", err)
		}

		if _, err := out.WriteString(fmt.Sprintln(fmt.Sprintf("# resolv.conf autogenerated by '%s'\n\nnameserver %s", os.Args[0], dnsCfg.Ip().String()))); err != nil {
			return fmt.Errorf("failed to change DNS configuration: %w", err)
		}

		if err := out.Sync(); err != nil {
			return fmt.Errorf("failed to change DNS configuration: %w", err)
		}
		return nil
	}

	_, err := createBackupIfNotExists()
	if err != nil {
		return err
	}

	// Save new configuration
	if err := saveNewConfig(); err != nil {
		return err
	}

	manualDNS = dnsCfg

	// enable file change monitoring
	go func() {
		w, err := fsnotify.NewWatcher()
		if err != nil {
			log.Error(fmt.Errorf("failed to start DNS-change monitoring (fsnotify error): %w", err))
			return
		}

		log.Info("DNS-change monitoring started")
		defer func() {
			log.Info("DNS-change monitoring stopped")
			w.Close()
		}()

		for {
			// start watching file
			err = w.Add(resolvFile)
			if err != nil {
				log.Error(fmt.Errorf("failed to start DNS-change monitoring (fsnotify error): %w", err))
				return
			}

			// wait for changes
			var evt fsnotify.Event
			select {
			case evt = <-w.Events:
			case <-done:
				// monitoring stopped
				return
			}

			//stop watching file
			if err := w.Remove(resolvFile); err != nil {
				log.Error(fmt.Errorf("failed to remove warcher (fsnotify error): %w", err))
			}

			// wait 2 seconds for reaction (in case if we are stopping of when multiple consecutive file changes)
			select {
			case <-time.After(time.Second * 2):
			case <-done:
				// monitoring stopped
				return
			}

			// restore DNS configuration
			log.Info(fmt.Sprintf("DNS-change monitoring: DNS was changed outside [%s]. Restoring ...", evt.Op.String()))
			if err := saveNewConfig(); err != nil {
				log.Error(err)
			}
		}
	}()

	return nil
}

// DeleteManual - reset manual DNS configuration to default
// 'localInterfaceIP' (obligatory only for Windows implementation) - local IP of VPN interface
func implDeleteManual(localInterfaceIP net.IP) error {
	if isPaused {
		// in case of PAUSED state -> just save manualDNS config
		// it will be applied on RESUME
		manualDNS = DnsSettings{}
		return nil
	}

	stopDnscryptProxyProcess()

	// stop file change monitoring
	stopDNSChangeMonitoring()
	isDeleteBackup := true // delete backup file
	return restoreBackup(resolvBackupFile, isDeleteBackup)
}

func implGetPredefinedDnsConfigurations() ([]DnsSettings, error) {
	return []DnsSettings{}, nil
}

func stopDNSChangeMonitoring() {
	// stop file change monitoring
	select {
	case done <- struct{}{}:
		break
	default:
		break
	}
}

func stopDnscryptProxyProcess() {
	if dnscryptProxyProcess != nil {
		dnscryptProxyProcess.Stop()
		dnscryptProxyProcess = nil
	}
}

func isBackupExists(backupFName string) bool {
	_, err := os.Stat(backupFName)
	return err == nil
}

func createBackup(backupFName string, isOverwriteIfExists bool) (created bool, er error) {
	if _, err := os.Stat(resolvFile); err != nil {
		// source file not exists
		return false, fmt.Errorf("failed to backup DNS configuration (file availability check failed): %w", err)
	}

	if _, err := os.Stat(backupFName); err == nil {
		// backup file already exists
		if !isOverwriteIfExists {
			return false, nil
		}
	}

	if err := os.Rename(resolvFile, backupFName); err != nil {
		return false, fmt.Errorf("failed to backup DNS configuration: %w", err)
	}
	return true, nil
}

func restoreBackup(backupFName string, isDeleteBackup bool) error {
	if _, err := os.Stat(backupFName); err != nil {
		// nothing to restore
		return nil
	}

	// restore original configuration
	if isDeleteBackup {
		if err := os.Rename(backupFName, resolvFile); err != nil {
			return fmt.Errorf("failed to restore DNS configuration: %w", err)
		}
	} else {
		tmpFName := resolvFile + ".tmp"
		if err := helpers.CopyFile(backupFName, tmpFName); err != nil {
			return fmt.Errorf("failed to restore DNS configuration: %w", err)
		}
		if err := os.Chmod(tmpFName, defaultFilePermissions); err != nil {
			return fmt.Errorf("failed to restore DNS configuration: %w", err)
		}
		if err := os.Rename(tmpFName, resolvFile); err != nil {
			return fmt.Errorf("failed to restore DNS configuration: %w", err)
		}
	}

	return nil
}
