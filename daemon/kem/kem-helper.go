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

//
// Wrapper to 'kem-helper' tool: https://github.com/ivpn/desktop-app/tree/master/daemon/References/common/kem-helper
//

package kem

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

type Kem_Algo_Name string

const (
	AlgName_Kyber1024             Kem_Algo_Name = "Kyber1024"
	AlgName_ClassicMcEliece348864 Kem_Algo_Name = "Classic-McEliece-348864"
)

type KemHelper struct {
	kemHelperPath string
	algorithms    []Kem_Algo_Name
	privateKeys   []string // base64
	publicKeys    []string // base64
	ciphers       []string // base64
	secrets       []string // base64 (decoded ciphers)
}

func GetDefaultKemAlgorithms() []Kem_Algo_Name {
	return []Kem_Algo_Name{AlgName_Kyber1024, AlgName_ClassicMcEliece348864}
}

// Initialise KEM helper and generate Key pairs
// IMPORTANT! The algorithms order in argument 'kemAlgorithms' is important! It in use for PresharedKey calculation!
func CreateHelper(kemHelperBinaryPath string, kemAlgorithms []Kem_Algo_Name) (*KemHelper, error) {
	if len(kemHelperBinaryPath) == 0 {
		return nil, fmt.Errorf("kem-helper error: bad argument (kem helper binary path not defined)")
	}
	if len(kemAlgorithms) == 0 {
		return nil, fmt.Errorf("kem-helper error: bad argument (kemAlgorithms not defined)")
	}
	helper := &KemHelper{
		kemHelperPath: kemHelperBinaryPath,
		ciphers:       make([]string, len(kemAlgorithms)),
		algorithms:    append([]Kem_Algo_Name{}, kemAlgorithms...)}

	if err := helper.generateKeys(); err != nil {
		return nil, err
	}

	return helper, nil
}

func (k KemHelper) GetPublicKey(kemAlgoName Kem_Algo_Name) (string, error) {
	idx, err := k.getAlgoIndex(kemAlgoName)
	if err != nil {
		return "", err
	}
	return k.publicKeys[idx], nil
}

func (k KemHelper) SetCipher(kemAlgoName Kem_Algo_Name, cipher string) error {
	idx, err := k.getAlgoIndex(kemAlgoName)
	if err != nil {
		return err
	}
	k.ciphers[idx] = cipher
	return nil
}

func (k *KemHelper) CalculatePresharedKey() (presharedKeyBase64 string, retErr error) {
	err := k.checkCiphers()
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

func (k KemHelper) getAlgoIndex(alg Kem_Algo_Name) (index int, err error) {
	for idx, a := range k.algorithms {
		if a == alg {
			return idx, nil
		}
	}
	return -1, fmt.Errorf("KEM algorithm `%s` not initialised", alg)
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

func (k *KemHelper) checkCiphers() error {
	var errSb strings.Builder
	for idx, alg := range k.algorithms {
		if len(k.ciphers) <= idx || len(k.ciphers[idx]) <= 0 {
			errSb.WriteString(string(alg) + ";")
		}
	}
	if errSb.Len() > 0 {
		return fmt.Errorf("ciphers are not defined: %s", errSb.String())
	}
	return nil
}
