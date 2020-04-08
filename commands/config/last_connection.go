package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

func lastConnectionInfoFile() string {
	dir, err := configDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "last_connection")
}

// LastConnectionInfo information about last connection parameters
type LastConnectionInfo struct {
	Gateway         string
	Port            string
	Obfsproxy       bool
	FirewallOff     bool
	DNS             string
	Antitracker     bool
	AntitrackerHard bool

	MultiopExitSvr string
}

// LastConnectionExist - returns 'true' if available info about last successful connection
func LastConnectionExist() bool {
	if _, err := os.Stat(lastConnectionInfoFile()); err == nil {
		return true
	}
	return false
}

// SaveLastConnectionInfo save last connection parameters in local storage
func SaveLastConnectionInfo(ci LastConnectionInfo) {
	data, err := json.Marshal(ci)
	if err != nil {
		return
	}

	if file := lastConnectionInfoFile(); len(file) > 0 {
		ioutil.WriteFile(file, data, 0600) // read only for owner
	}
}

// RestoreLastConnectionInfo restore last connection info from local storage
func RestoreLastConnectionInfo() *LastConnectionInfo {
	ci := LastConnectionInfo{}

	file := ""
	if file = lastConnectionInfoFile(); len(file) == 0 {
		return nil
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	err = json.Unmarshal(data, &ci)
	if err != nil {
		return nil
	}

	return &ci
}
