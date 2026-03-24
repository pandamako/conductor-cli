package config

import (
	"os"
	"path/filepath"
)

func DefaultConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".conductor-cli")
}

func DefaultConfigPath() string {
	return filepath.Join(DefaultConfigDir(), "config.json")
}
