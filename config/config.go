package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

type Config struct {
	ClientId string   `json:"client_id"`
	Emails   []string `json:"emails"`
}

func getConfigPath() (string, error) {
	var configDir string
	var err error

	if runtime.GOOS == "windows" {
		configDir, err = os.UserConfigDir()
	} else {
		configDir, err = os.UserHomeDir()
		if err == nil {
			configDir = filepath.Join(configDir, ".config")
		}
	}

	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "grabotp", "config.json"), nil
}

func ReadConfig() (Config, error) {
	path, err := getConfigPath()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}

		return Config{}, err
	}

	var config Config
	err = json.Unmarshal(data, &config)

	return config, err
}

func WriteConfig(config Config) error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
