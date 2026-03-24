package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"conductor-cli/internal/config"
)

// setupTestRepo creates a temp git repo with an initial commit.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "initial"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s", args, out)
		}
	}
	return dir
}

// withConfig overrides configPath for a test and returns the temp config path.
func withConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	orig := configPath
	configPath = path
	t.Cleanup(func() { configPath = orig })
	return path
}

// resolveSymlinks resolves macOS /private/var vs /var differences.
func resolveSymlinks(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return resolved
}

func TestInitCommand(t *testing.T) {
	cfgPath := withConfig(t)
	repo := setupTestRepo(t)

	cmd := &InitCommand{}
	if err := cmd.execute(repo); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	repoReal := resolveSymlinks(repo)
	found := false
	for path := range cfg.Repositories {
		if resolveSymlinks(path) == repoReal {
			found = true
			break
		}
	}
	if !found {
		t.Error("repository not found in config")
	}

	conductorDir := filepath.Join(repo, ".conductor-cli")
	if _, err := os.Stat(conductorDir); os.IsNotExist(err) {
		t.Error(".conductor-cli/ directory not created")
	}
}

func TestInitCommandNotGitRepo(t *testing.T) {
	withConfig(t)
	dir := t.TempDir()

	cmd := &InitCommand{}
	if err := cmd.execute(dir); err == nil {
		t.Error("expected error for non-git directory")
	}
}

func TestInitCommandAlreadyRegistered(t *testing.T) {
	withConfig(t)
	repo := setupTestRepo(t)

	cmd := &InitCommand{}
	if err := cmd.execute(repo); err != nil {
		t.Fatalf("first init failed: %v", err)
	}
	if err := cmd.execute(repo); err != nil {
		t.Fatalf("second init should succeed (idempotent): %v", err)
	}
}

func TestInitCommandNameCollision(t *testing.T) {
	withConfig(t)

	// Create two repos with the same basename "api"
	base1 := filepath.Join(t.TempDir(), "org1")
	base2 := filepath.Join(t.TempDir(), "org2")
	repo1 := filepath.Join(base1, "api")
	repo2 := filepath.Join(base2, "api")

	for _, repo := range []string{repo1, repo2} {
		os.MkdirAll(repo, 0755)
		for _, args := range [][]string{
			{"git", "init"},
			{"git", "config", "user.email", "test@test.com"},
			{"git", "config", "user.name", "Test"},
			{"git", "commit", "--allow-empty", "-m", "initial"},
		} {
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Dir = repo
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%v in %s failed: %s", args, repo, out)
			}
		}
	}

	cmd1 := &InitCommand{}
	if err := cmd1.execute(repo1); err != nil {
		t.Fatalf("first init failed: %v", err)
	}

	cmd2 := &InitCommand{}
	if err := cmd2.execute(repo2); err == nil {
		t.Error("expected name collision error")
	}
}

func TestInitCommandCustomName(t *testing.T) {
	cfgPath := withConfig(t)
	repo := setupTestRepo(t)

	cmd := &InitCommand{Name: "custom-name"}
	if err := cmd.execute(repo); err != nil {
		t.Fatalf("init with --name failed: %v", err)
	}

	cfg, _ := config.Load(cfgPath)
	for _, r := range cfg.Repositories {
		if r.Name == "custom-name" {
			return
		}
	}
	t.Error("custom name not found in config")
}
