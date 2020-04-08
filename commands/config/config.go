package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

var _mutex sync.Mutex

// Configuration - local configuration
type Configuration struct {
	CustomDNS           string
	Antitracker         bool
	AntitrackerHardcore bool
}

// configDir is the path to configuration dirrectory
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(home, ".ivpn")
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return "", err
	}
	return dir, nil
}

func filePath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", fmt.Errorf("unable to find configuration folder: %w", err)
	}
	return filepath.Join(dir, "config"), nil
}

// SaveConfig saves configuration to local storage
func SaveConfig(conf Configuration) error {
	_mutex.Lock()
	defer _mutex.Unlock()

	data, err := json.Marshal(conf)
	if err != nil {
		return err
	}

	file, err := filePath()
	if err != nil {
		return fmt.Errorf("error determining configuration file location: %w", err)
	}
	return ioutil.WriteFile(file, data, 0600) // read only for owner
}

// GetConfig returns local configuration
func GetConfig() (Configuration, error) {
	_mutex.Lock()
	defer _mutex.Unlock()

	conf := Configuration{}

	file, err := filePath()
	if err != nil {
		return conf, fmt.Errorf("error determining configuration file destination: %w", err)
	}

	_, err = os.Stat(file)
	if os.IsNotExist(err) {
		return conf, nil
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return conf, fmt.Errorf("failed to read configuration: %w", err)
	}

	err = json.Unmarshal(data, &conf)
	if err != nil {
		return conf, fmt.Errorf("failed to deserialize configuration: %w", err)
	}

	return conf, nil
}
