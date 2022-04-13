//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2022 Privatus Limited.
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

package eap

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ivpn/desktop-app/daemon/helpers"
	"github.com/ivpn/desktop-app/daemon/service/platform/filerights"
)

type Eap struct {
	mutex              sync.Mutex
	secretFile         string
	lastFailedAttempts []time.Time
}

//  key for encryption secret string (just to make it a little bit harder to read the file for humans).
const secretEncryptionKey = "Fx>PT/*fllA3yr3}Jn+k(?h<~4%lJm$Y"

func Init(secretFile string) *Eap {
	return &Eap{secretFile: secretFile}
}

func (e *Eap) Secret() (retSecret string, retErr error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.doSecret()
}

func (e *Eap) IsEnabled() bool {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.doIsEnabled()
}

func (e *Eap) ForceDisable() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.doForceDisable()
}

func (e *Eap) SetSecret(oldSecret, newSecret string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.doSetSecret(oldSecret, newSecret)
}

func (e *Eap) CheckSecret(secretToCheck string) (retVal bool, err error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.doCheckSecret(secretToCheck)
}

// --------- private functions ---------

func (e *Eap) doSecret() (retSecret string, retErr error) {
	file := e.secretFile
	if len(file) <= 0 {
		return "", nil // paranoid mode not implemented for this platform
	}

	f, err := os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			e.lastFailedAttempts = nil
			return "", nil // paranoid mode disabled
		}
		return "", fmt.Errorf("the Enhanced App Protection file open error : %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("the Enhanced App Protection file status check error : %w", err)
	}

	// check file access rights
	//mode := stat.Mode()
	//expectedMode := os.FileMode(0600) // read only for privilaged user
	//if mode != expectedMode {
	//	p.paranoidModeForceDisable()
	//	return "", fmt.Errorf(fmt.Sprintf("the Enhanced App Protection file has wrong access permissions (%o but expected %o)", mode, expectedMode))
	//}

	// read file
	if stat.Size() > 1024*5 {
		return "", fmt.Errorf("the Enhanced App Protection file too big")
	}
	buff := make([]byte, stat.Size())
	_, err = f.Read(buff)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read Enhanced App Protection file: %w", err)
	}

	// decode
	data, err := helpers.DecryptString([]byte(secretEncryptionKey), string(buff))
	if err != nil {
		return "", fmt.Errorf("failed to decode EAP secret: %w", err)
	}

	// check first line
	lines := strings.Split(data, "\n")
	if len(lines) != 1 {
		return "", fmt.Errorf("wrong data in Enhanced App Protection file (expected one line)")
	}
	secret := strings.TrimSpace(lines[0])
	if len(secret) <= 0 {
		return "", fmt.Errorf("wrong data in Enhanced App Protection file (secret is empty)")
	}

	return secret, nil
}

func (e *Eap) doIsEnabled() bool {
	secret, err := e.doSecret()
	return err == nil && len(secret) > 0
}

func (e *Eap) doForceDisable() error {
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
		return fmt.Errorf("failed to disable Enhanced App Protection: %w", removeErr)
	}
	e.lastFailedAttempts = nil
	return nil
}

func (e *Eap) doSetSecret(oldSecret, newSecret string) error {
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
			return fmt.Errorf("the current shared secret for EAP does not match")
		}
	}

	if strings.TrimSpace(newSecret) != newSecret {
		return fmt.Errorf("please avoid using space symbols in EAP secret")
	}

	if len(strings.Split(newSecret, "\n")) != 1 {
		return fmt.Errorf("new shared secret for EAP should contain only one line")
	}

	if len(newSecret) == 0 {
		if isPmEnabled {
			// disable paranoid mode
			if err := e.doForceDisable(); err != nil {
				return fmt.Errorf("failed to disable EAP: %w", err)
			}
		}
		return nil
	}

	encrypted, err := helpers.EncryptString([]byte(secretEncryptionKey), newSecret)
	if err != nil {
		return fmt.Errorf("failed to encode EAP secret: %w", err)
	}

	bytesToWrite := []byte(encrypted)
	if len(bytesToWrite) > 1024*10 {
		return fmt.Errorf("password too long")
	}

	// save data
	if err := os.WriteFile(file, bytesToWrite, 0600); err != nil {
		e.doForceDisable()
		return fmt.Errorf("failed to enable EAP (FileWrite error): %w", err)
	}
	// only for Windows: Golang is not able to change file permissins in Windows style
	if err := filerights.WindowsChmod(file, 0600); err != nil { // read\write only for privileged user
		e.doForceDisable()
		return fmt.Errorf("failed to change EAP file permissions: %w", err)
	}

	// ensure data were saved correctly
	secretConfirm, err := e.doSecret()
	if err != nil {
		e.doForceDisable()
		return fmt.Errorf("failed to enable EAP: %w", err)
	}
	if secretConfirm != newSecret {
		e.doForceDisable()
		return fmt.Errorf("failed to enable EAP: internal error during confirmation")
	}

	return nil
}

func (e *Eap) doCheckSecret(secretToCheck string) (retVal bool, err error) {
	// some protection from brute force attack
	defer func() {
		if retVal {
			e.lastFailedAttempts = nil
		} else {
			e.lastFailedAttempts = append(e.lastFailedAttempts, time.Now())
		}
	}()

	// read secret
	secret, err := e.doSecret()
	isPModeEnabled := err == nil && len(secret) > 0
	if !isPModeEnabled {
		return true, nil
	}

	// some protection from brute force attack
	const maxFailAttemptsCnt = 5
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

	if cntAttempts > 2 {
		// There is possibility of unexpected manipulation with system time.
		// We mitigate it a little: perform 1 second delay if there are more than 2 failed requests
		// (independently from system time)
		time.Sleep(time.Second)
	}

	// compare secrets
	return secret == secretToCheck, nil
}
