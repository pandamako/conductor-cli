package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type RepoConfig struct {
	Name string `json:"name"`
}

type GlobalConfig struct {
	WorktreesBasePath string                `json:"worktrees_base_path"`
	Repositories      map[string]RepoConfig `json:"repositories"`
}

func DefaultConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".conductor-cli")
}

func DefaultConfigPath() string {
	return filepath.Join(DefaultConfigDir(), "config.json")
}

func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	if path == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	return path
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

func Load(path string) (*GlobalConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &GlobalConfig{
				WorktreesBasePath: filepath.Join(DefaultConfigDir(), "worktrees"),
				Repositories:      make(map[string]RepoConfig),
			}, nil
		}
		return nil, err
	}
	var cfg GlobalConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	cfg.WorktreesBasePath = expandTilde(cfg.WorktreesBasePath)
	if cfg.Repositories == nil {
		cfg.Repositories = make(map[string]RepoConfig)
	}
	return &cfg, nil
}

func Save(path string, cfg *GlobalConfig) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (c *GlobalConfig) WorktreePath(repoName, branchName string) string {
	return filepath.Join(c.WorktreesBasePath, repoName, SanitizeBranchName(branchName))
}

func (c *GlobalConfig) FindRepoByName(name string) (string, bool) {
	for path, repo := range c.Repositories {
		if repo.Name == name {
			return path, true
		}
	}
	return "", false
}
