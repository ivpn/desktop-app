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
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type genkeysResp struct {
	Priv string `json:"priv"`
	Pub  string `json:"pub"`
}
type decpskArgs struct {
	Cipher string `json:"cipher"`
	Priv   string `json:"priv"`
}
type decpskResp struct {
	Secret string `json:"secret"`
}

func GenerateKeys(kemHelperPath string, kemAlgorithmName Kem_Algo_Name) (privateKeyBase64, publicKeyBase64 string, retErr error) {
	if string(kemAlgorithmName) == "" {
		return "", "", fmt.Errorf("kem-helper error: bad argument")
	}
	cmd := exec.Command(kemHelperPath, "genkeys", string(kemAlgorithmName))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("kem-helper error (kem:%s): %w (%s)", kemAlgorithmName, err, strings.TrimSpace(stderr.String()))
	}

	resp := genkeysResp{}
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return "", "", fmt.Errorf("kem-helper error: %w", err)
	}

	if len(resp.Priv) == 0 || len(resp.Pub) == 0 {
		return "", "", fmt.Errorf("kem-helper error: empty keys")
	}

	return resp.Priv, resp.Pub, nil
}

func GenerateKeysMulti(kemHelperPath string, kemAlgorithmName []Kem_Algo_Name) (privateKeyBase64, publicKeyBase64 []string, retErr error) {
	if len(kemAlgorithmName) == 0 {
		return nil, nil, fmt.Errorf("kem-helper error: bad argument")
	}

	for _, alg := range kemAlgorithmName {
		priv, public, err := GenerateKeys(kemHelperPath, alg)
		if err != nil {
			return []string{}, []string{}, err
		}
		privateKeyBase64 = append(privateKeyBase64, priv)
		publicKeyBase64 = append(publicKeyBase64, public)
	}

	return privateKeyBase64, publicKeyBase64, nil
}

func DecodeCipher(kemHelperPath string, kemAlgorithmName Kem_Algo_Name, privateKeyBase64 string, cipherBase64 string) (secretBase64 string, retErr error) {
	if string(kemAlgorithmName) == "" {
		return "", fmt.Errorf("kem-helper error: bad argument")
	}

	data := decpskArgs{Cipher: cipherBase64, Priv: privateKeyBase64}
	dataJsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("kem-helper error: %w", err)
	}

	cmd := exec.Command(kemHelperPath, "decpsk", string(kemAlgorithmName))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = strings.NewReader(string(dataJsonBytes))

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("kem-helper error (kem:%s): %w (%s)", kemAlgorithmName, err, strings.TrimSpace(stderr.String()))
	}

	resp := decpskResp{}
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return "", fmt.Errorf("kem-helper error (kem:%s): %w", kemAlgorithmName, err)
	}
	if len(resp.Secret) == 0 {
		return "", fmt.Errorf("kem-helper error: empty secret")
	}
	return resp.Secret, nil
}

func DecodeCipherMulti(kemHelperPath string, kemAlgorithmName []Kem_Algo_Name, privateKeyBase64 []string, cipherBase64 []string) (secretBase64 []string, retErr error) {
	chipersCnt := len(cipherBase64)
	algsCnt := len(kemAlgorithmName)
	if algsCnt == 0 || algsCnt != chipersCnt || chipersCnt != len(privateKeyBase64) {
		return []string{}, fmt.Errorf("kem-helper error: bad argument")
	}

	for i := range kemAlgorithmName {
		secret, err := DecodeCipher(kemHelperPath, kemAlgorithmName[i], privateKeyBase64[i], cipherBase64[i])
		if err != nil {
			return []string{}, err
		}
		secretBase64 = append(secretBase64, secret)
	}
	return secretBase64, nil
}

/*
type encpskArgs struct {
	Pub string `json:"pub"`
}
type encpskResp struct {
	Cipher string `json:"cipher"`
	Secret string `json:"secret"`
}

func EncodeCipher(kemHelperPath string, kemAlgorithmName kem_Algo_Name, publicKeyBase64 string) (secretBase64, cipherBase64 string, retErr error) {
	if string(kemAlgorithmName) == "" {
		return "", "", fmt.Errorf("kem-helper error: bad argument")
	}

	data := encpskArgs{Pub: publicKeyBase64}
	dataJsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", "", fmt.Errorf("kem-helper error: %w", err)
	}

	cmd := exec.Command(kemHelperPath, "encpsk", string(kemAlgorithmName))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = strings.NewReader(string(dataJsonBytes))

	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("kem-helper error: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}

	resp := encpskResp{}
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return "", "", fmt.Errorf("kem-helper error: %w", err)
	}
	if len(resp.Secret) == 0 || len(resp.Cipher) == 0 {
		return "", "", fmt.Errorf("kem-helper error: empty secret/cipher")
	}
	return resp.Secret, resp.Cipher, nil
}

func EncodeCipherMulti(kemHelperPath string, kemAlgorithmName []kem_Algo_Name, publicKeyBase64 []string) (secretBase64, cipherBase64 []string, retErr error) {
	algsCnt := len(kemAlgorithmName)
	if algsCnt == 0 || algsCnt != len(publicKeyBase64) {
		return []string{}, []string{}, fmt.Errorf("kem-helper error: bad argument")
	}

	for i := range kemAlgorithmName {
		secret, cipher, err := EncodeCipher(kemHelperPath, kemAlgorithmName[i], publicKeyBase64[i])
		if err != nil {
			return []string{}, []string{}, err
		}
		secretBase64 = append(secretBase64, secret)
		cipherBase64 = append(cipherBase64, cipher)
	}
	return secretBase64, cipherBase64, nil
}
*/
