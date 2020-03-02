package wireguard

import (
	"io"
	"os/exec"
)

// GenerateKeys generates new WireGuard keys pair
func GenerateKeys(wgToolBinaryPath string) (publicKey string, privateKey string, err error) {
	// private key
	privCmd := exec.Command(wgToolBinaryPath, "genkey")
	out, err1 := privCmd.Output()
	if err1 != nil {
		return "", "", err1
	}
	privateKey = string(out)

	// public key
	pubCmd := exec.Command(wgToolBinaryPath, "pubkey")
	stdin, err2 := pubCmd.StdinPipe()
	if err2 != nil {
		return "", "", err2
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, privateKey) // write to command stdin
	}()

	out, err = pubCmd.Output()
	if err != nil {
		return "", "", err
	}
	publicKey = string(out)

	return publicKey, privateKey, nil
}
