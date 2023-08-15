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

package eaa

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ivpn/desktop-app/daemon/helpers"
)

// Enhanced App Authentication
type Eaa struct {
	mutex              sync.Mutex
	secretFile         string
	lastFailedAttempts []time.Time
}

func Init(secretFile string) *Eaa {
	return &Eaa{secretFile: secretFile}
}

func (e *Eaa) IsEnabled() bool {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.doIsEnabled()
}

func (e *Eaa) ForceDisable() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.doForceDisable()
}

func (e *Eaa) SetSecret(oldSecret, newSecret string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.doSetSecret(oldSecret, newSecret)
}

func (e *Eaa) CheckSecret(secretToCheck string) (retVal bool, err error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.doCheckSecret(secretToCheck)
}

// --------- private functions ---------

func (e *Eaa) doGetSecretHash() (retSecretHash []byte, retErr error) {
	file := e.secretFile
	if len(file) <= 0 {
		return nil, nil // paranoid mode not implemented for this platform
	}

	f, err := os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			e.lastFailedAttempts = nil
			return nil, nil // paranoid mode disabled
		}
		return nil, fmt.Errorf("the EAA file open error : %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("the EAA file status check error : %w", err)
	}

	// check file access rights
	//mode := stat.Mode()
	//expectedMode := os.FileMode(0600) // read only for privilaged user
	//if mode != expectedMode {
	//	p.paranoidModeForceDisable()
	//	return "", fmt.Errorf(fmt.Sprintf("the EAA file has wrong access permissions (%o but expected %o)", mode, expectedMode))
	//}

	// read file
	if stat.Size() > 1024*5 {
		return nil, fmt.Errorf("the EAA file too big")
	}
	buff := make([]byte, stat.Size())
	_, err = f.Read(buff)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read EAA file: %w", err)
	}

	return buff, nil
}

func (e *Eaa) doIsEnabled() bool {
	secretHash, err := e.doGetSecretHash()
	return err == nil && len(secretHash) > 0
}

func (e *Eaa) doForceDisable() error {
	file := e.secretFile
	if len(file) <= 0 {
		return nil // paranoid mode not implemented for this platform
	}

	// Disable paranoid mode (remove secret file)
	// In case of error - do additional retries with small delay (3 times)
	var removeErr error
	for i := 0; i < 3; i++ {
		removeErr = os.Remove(file)
		if removeErr == nil {
			break
		}
		time.Sleep(time.Millisecond * 50)
	}

	if removeErr != nil {
		return fmt.Errorf("failed to disable EAA: %w", removeErr)
	}
	e.lastFailedAttempts = nil
	return nil
}

func (e *Eaa) doSetSecret(oldSecret, newSecret string) error {
	file := e.secretFile
	if len(file) <= 0 {
		return nil // paranoid mode not implemented for this platform
	}

	isPmEnabled := e.doIsEnabled()
	if isPmEnabled {
		// we MUST call 'doCheckSecret()' because it has protection from brute force attack
		isOK, err := e.doCheckSecret(oldSecret)
		if err != nil {
			return err
		}
		if isPmEnabled && !isOK {
			return fmt.Errorf("the current password for EAA does not match")
		}
	}

	if strings.TrimSpace(newSecret) != newSecret {
		return fmt.Errorf("please avoid using space symbols in EAA password")
	}

	if len(newSecret) == 0 {
		if isPmEnabled {
			// disable paranoid mode
			if err := e.doForceDisable(); err != nil {
				return fmt.Errorf("failed to disable EAA: %w", err)
			}
		}
		return nil
	}

	// save data
	if err := helpers.WriteFile(file, []byte(newSecret), 0600); err != nil {
		e.doForceDisable()
		return fmt.Errorf("failed to enable EAA (FileWrite error): %w", err)
	}

	// ensure data were saved correctly
	isOK, err := e.doCheckSecret(newSecret)
	if err != nil {
		e.doForceDisable()
		return fmt.Errorf("failed to enable EAA: %w", err)
	}
	if !isOK {
		e.doForceDisable()
		return fmt.Errorf("failed to enable EAA: internal error during confirmation")
	}

	return nil
}

func (e *Eaa) doCheckSecret(secretToCheck string) (retVal bool, err error) {
	// some protection from brute force attack
	defer func() {
		if retVal {
			e.lastFailedAttempts = nil
		} else {
			e.lastFailedAttempts = append(e.lastFailedAttempts, time.Now())
		}
	}()

	// read secretHash
	secretHash, err := e.doGetSecretHash()
	isPModeEnabled := err == nil && len(secretHash) > 0
	if !isPModeEnabled {
		return true, nil
	}

	// some protection from brute force attack
	const maxFailAttemptsCnt = 6
	const maxFailDuration = time.Minute

	cntAttempts := len(e.lastFailedAttempts)
	if cntAttempts >= maxFailAttemptsCnt {
		if cntAttempts > maxFailAttemptsCnt {
			// trim array: get last "maxFailAttemptsCnt" elements
			e.lastFailedAttempts = e.lastFailedAttempts[cntAttempts-maxFailAttemptsCnt:]
		}

		if e.lastFailedAttempts[0].Add(maxFailDuration).After(time.Now()) {
			return false, fmt.Errorf("You have exceeded the allowed number of requests. Please wait 1 minute and try again.")
		}
	}

	if cntAttempts > 4 {
		// There is possibility of unexpected manipulation with system time.
		// We mitigate it a little: perform 1 second delay if there are more than 2 failed requests
		// (independently from system time)
		time.Sleep(time.Second)
	}

	// Compare secrets
	// (password hashing is done on client side. So we just comparing data.)
	return bytes.Equal(secretHash, []byte(secretToCheck)), nil
}
