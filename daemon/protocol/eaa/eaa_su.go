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

package eaa

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/ivpn/desktop-app/daemon/helpers"
)

// Enhanced App Authentication for a clients running in privilaged environment
type EaaSu struct {
	mutex      sync.Mutex
	secret     []byte
	secretFile string
}

const (
	secretLen = 64
)

func InitializeSuAccess(suFilesFolder string, connectionName string) (*EaaSu, error) {
	if len(suFilesFolder) <= 0 {
		return nil, fmt.Errorf("folder not specified")
	}

	secret := make([]byte, secretLen)
	if err := binary.Read(rand.Reader, binary.BigEndian, secret); err != nil {
		return nil, fmt.Errorf("failed to generete secret: %w", err)
	}

	if err := os.MkdirAll(suFilesFolder, 0600); err != nil {
		return nil, err
	}

	file := path.Join(suFilesFolder, connectionName)
	if err := helpers.WriteFile(file, secret, 0600); err != nil {
		return nil, fmt.Errorf("failed to create EAA SU file: %w", err)
	}

	return &EaaSu{secret: secret, secretFile: file}, nil
}

func (s *EaaSu) File() string {
	return s.secretFile
}

func (s *EaaSu) IsInitialized() bool {
	return len(s.secretFile) > 0
}

func (s *EaaSu) CheckSecret(secret []byte) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.secret) == 0 {
		return fmt.Errorf("not initialized")
	}
	isOK := bytes.Equal(s.secret, s.secret)
	if !isOK {
		s.doUnInitialize()
		return fmt.Errorf("secret does not match")
	}
	return nil
}

func (s *EaaSu) UnInitialize() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.doUnInitialize()
}

// --------- private functions ---------

func (s *EaaSu) doUnInitialize() error {
	s.secret = nil

	if len(s.secretFile) > 0 {
		if err := os.Remove(s.secretFile); err != nil {
			return fmt.Errorf("unable to remove file: %w", err)
		}
	}
	s.secretFile = ""

	return nil
}
