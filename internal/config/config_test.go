package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"main", "main"},
		{"feature/login", "feature/login"},
		{"feature/deep/nested", "feature/deep/nested"},
		{"has%percent", "has%percent"},
		{"feature%2Flogin", "feature%2Flogin"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SanitizeBranchName(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeBranchName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestLoadNonExistent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent", "config.json")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Repositories == nil || len(cfg.Repositories) != 0 {
		t.Error("expected empty repositories map")
	}
	home, _ := os.UserHomeDir()
	expectedBase := filepath.Join(home, ".conductor-cli", "worktrees")
	if cfg.WorktreesBasePath != expectedBase {
		t.Errorf("WorktreesBasePath = %q, want %q", cfg.WorktreesBasePath, expectedBase)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := &GlobalConfig{
		WorktreesBasePath: "/custom/path",
		Repositories: map[string]RepoConfig{
			"/home/user/repo": {Name: "repo"},
		},
	}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.WorktreesBasePath != "/custom/path" {
		t.Errorf("WorktreesBasePath = %q, want %q", loaded.WorktreesBasePath, "/custom/path")
	}
	repo, ok := loaded.Repositories["/home/user/repo"]
	if !ok {
		t.Fatal("expected repository entry")
	}
	if repo.Name != "repo" {
		t.Errorf("repo.Name = %q, want %q", repo.Name, "repo")
	}
}

func TestTildeExpansion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	home, _ := os.UserHomeDir()

	cfg := &GlobalConfig{
		WorktreesBasePath: "~/my-worktrees",
		Repositories:      map[string]RepoConfig{},
	}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	expected := filepath.Join(home, "my-worktrees")
	if loaded.WorktreesBasePath != expected {
		t.Errorf("WorktreesBasePath = %q, want %q", loaded.WorktreesBasePath, expected)
	}
}

func TestFindRepoByName(t *testing.T) {
	cfg := &GlobalConfig{
		Repositories: map[string]RepoConfig{
			"/path/a": {Name: "alpha"},
			"/path/b": {Name: "beta"},
		},
	}

	path, found := cfg.FindRepoByName("alpha")
	if !found || path != "/path/a" {
		t.Errorf("FindRepoByName('alpha') = %q, %v", path, found)
	}

	_, found = cfg.FindRepoByName("gamma")
	if found {
		t.Error("expected not found for 'gamma'")
	}
}

func TestWorktreePath(t *testing.T) {
	cfg := &GlobalConfig{WorktreesBasePath: "/base"}
	got := cfg.WorktreePath("myrepo", "feature/login")
	expected := filepath.Join("/base", "myrepo", "feature", "login")
	if got != expected {
		t.Errorf("WorktreePath = %q, want %q", got, expected)
	}
}
