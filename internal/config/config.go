package config

import (
	"os"
	"path/filepath"
	"strings"
)

func DefaultConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".conductor-cli")
}

func DefaultConfigPath() string {
	return filepath.Join(DefaultConfigDir(), "config.json")
}

// SanitizeBranchName encodes a branch name for use as a directory name.
// Percent signs are encoded first (%25), then slashes (%2F).
func SanitizeBranchName(name string) string {
	name = strings.ReplaceAll(name, "%", "%25")
	name = strings.ReplaceAll(name, "/", "%2F")
	return name
}

// UnsanitizeBranchName reverses SanitizeBranchName.
func UnsanitizeBranchName(name string) string {
	name = strings.ReplaceAll(name, "%2F", "/")
	name = strings.ReplaceAll(name, "%25", "%")
	return name
}
