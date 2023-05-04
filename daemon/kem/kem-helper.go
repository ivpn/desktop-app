//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 Privatus Limited.
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

//
// Wrapper to 'kem-helper' tool: https://github.com/ivpn/desktop-app/tree/master/daemon/References/common/kem-helper
//

package kem

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

type kem_Algo_Name string

const (
	kem_Algo_Name_Kyber1024             kem_Algo_Name = "Kyber1024"
	kem_Algo_Name_ClassicMcEliece348864 kem_Algo_Name = "Classic-McEliece-348864"
)

type KemHelper struct {
	kemHelperPath string
	algorithms    []kem_Algo_Name
	privateKeys   []string // base64
	publicKeys    []string // base64
	ciphers       []string // base64
	secrets       []string // base64 (decoded ciphers)
}

// Initialise KEM helper and generate Key pairs
func CreateHelper(kemHelperBinaryPath string) (*KemHelper, error) {
	if len(kemHelperBinaryPath) == 0 {
		return nil, fmt.Errorf("kem-helper error: bad argument (kem helper binary path not defined)")
	}
	helper := &KemHelper{
		kemHelperPath: kemHelperBinaryPath,
		ciphers:       []string{"", ""},
		algorithms: []kem_Algo_Name{
			kem_Algo_Name_Kyber1024,
			kem_Algo_Name_ClassicMcEliece348864}}

	if err := helper.generateKeys(); err != nil {
		return nil, err
	}

	return helper, nil
}

func (k KemHelper) GetPublicKey_Kyber1024() string {
	return k.publicKeys[0]
}
func (k KemHelper) GetPublicKey_ClassicMcEliece348864() string {
	return k.publicKeys[1]
}
func (k *KemHelper) SetCipher_Kyber1024(c string) {
	k.ciphers[0] = c
}
func (k *KemHelper) SetCipher_ClassicMcEliece348864(c string) {
	k.ciphers[1] = c
}
func (k *KemHelper) IsNil() bool {
	return k == nil
}

func (k *KemHelper) CheckCiphers() error {
	hasCipher_Kyber1024 := len(k.ciphers[0]) > 0
	hasCipher_ClassicMcEliece348864 := len(k.ciphers[1]) > 0
	if !hasCipher_Kyber1024 || !hasCipher_ClassicMcEliece348864 {
		return fmt.Errorf("ciphers are not defined: cipher1=%v; cipher2=%v",
			hasCipher_Kyber1024, hasCipher_ClassicMcEliece348864)
	}
	return nil
}

func (k *KemHelper) CalculatePresharedKey() (presharedKeyBase64 string, retErr error) {
	err := k.CheckCiphers()
	if err != nil {
		return "", fmt.Errorf("KemHelper error: %w", err)
	}
	if err := k.decodeCiphers(k.ciphers); err != nil {
		return "", err
	}
	if len(k.privateKeys) == 0 {
		return "", fmt.Errorf("KemHelper error: private keys not defined")
	}
	if len(k.secrets) == 0 {
		return "", fmt.Errorf("KemHelper error: secrets not defined")
	}

	hasher := sha256.New()
	for i := range k.secrets {
		sDecoded, err := base64.StdEncoding.DecodeString(k.secrets[i])
		if err != nil {
			return "", fmt.Errorf("KemHelper error: %w", err)
		}
		hasher.Write(sDecoded)
	}

	return base64.StdEncoding.EncodeToString(hasher.Sum(nil)), nil
}

func (k *KemHelper) generateKeys() (retErr error) {
	k.privateKeys, k.publicKeys, retErr = GenerateKeysMulti(k.kemHelperPath, k.algorithms)
	return retErr
}

func (k *KemHelper) decodeCiphers(cipherBase64 []string) (retErr error) {
	if len(k.algorithms) != len(cipherBase64) {
		return fmt.Errorf("KemHelper error: unexpected count of ciphers to decode")
	}
	k.secrets, retErr = DecodeCipherMulti(k.kemHelperPath, k.algorithms, k.privateKeys, cipherBase64)
	if retErr != nil {
		k.secrets = []string{}
	}
	return retErr
}
