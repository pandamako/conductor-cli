package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"conductor-cli/internal/config"
	"conductor-cli/internal/git"
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
// Pre-seeds config with a temp WorktreesBasePath so tests don't pollute ~/.conductor-cli/.
// Tests using this helper must NOT call t.Parallel(), as configPath is shared.
func withConfig(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")
	// Pre-seed with temp worktrees base path
	cfg := &config.GlobalConfig{
		WorktreesBasePath: filepath.Join(tmpDir, "worktrees"),
		Repositories:      map[string]config.RepoConfig{},
	}
	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("failed to seed config: %v", err)
	}
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

func TestCreateCommand(t *testing.T) {
	withConfig(t)
	repo := setupTestRepo(t)

	initCmd := &InitCommand{}
	if err := initCmd.execute(repo); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	createCmd := &CreateCommand{}
	wtPath, err := createCmd.execute(repo, "test-branch")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Error("worktree directory not created")
	}

	worktrees, err := git.WorktreeList(repo)
	if err != nil {
		t.Fatalf("WorktreeList() error = %v", err)
	}
	found := false
	for _, wt := range worktrees {
		if wt.Branch == "test-branch" {
			found = true
			break
		}
	}
	if !found {
		t.Error("branch 'test-branch' not found in worktree list")
	}
}

func TestCreateCommandWithSlash(t *testing.T) {
	withConfig(t)
	repo := setupTestRepo(t)

	initCmd := &InitCommand{}
	if err := initCmd.execute(repo); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	createCmd := &CreateCommand{}
	wtPath, err := createCmd.execute(repo, "feature/login")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Error("worktree directory not created")
	}
	dir := filepath.Base(wtPath)
	if dir != "feature%2Flogin" {
		t.Errorf("directory name = %q, want %q", dir, "feature%2Flogin")
	}
}

func TestCreateCommandNotRegistered(t *testing.T) {
	withConfig(t)
	repo := setupTestRepo(t)

	createCmd := &CreateCommand{}
	_, err := createCmd.execute(repo, "test-branch")
	if err == nil {
		t.Error("expected error for unregistered repo")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreateCommandWithSetupScript(t *testing.T) {
	withConfig(t)
	repo := setupTestRepo(t)

	initCmd := &InitCommand{}
	if err := initCmd.execute(repo); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	setupScript := filepath.Join(repo, ".conductor-cli", "setup")
	os.WriteFile(setupScript, []byte("#!/bin/sh\ntouch setup-was-run\n"), 0755)

	createCmd := &CreateCommand{}
	wtPath, err := createCmd.execute(repo, "with-setup")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	marker := filepath.Join(wtPath, "setup-was-run")
	if _, err := os.Stat(marker); os.IsNotExist(err) {
		t.Error("setup script was not executed in worktree directory")
	}
}

func TestCreateCommandSetupNotExecutable(t *testing.T) {
	withConfig(t)
	repo := setupTestRepo(t)

	initCmd := &InitCommand{}
	if err := initCmd.execute(repo); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	setupScript := filepath.Join(repo, ".conductor-cli", "setup")
	os.WriteFile(setupScript, []byte("#!/bin/sh\necho hi\n"), 0644)

	createCmd := &CreateCommand{}
	_, err := createCmd.execute(repo, "no-exec")
	if err == nil {
		t.Error("expected error for non-executable setup script")
	}
	if !strings.Contains(err.Error(), "not executable") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreateCommandSetupScriptFails(t *testing.T) {
	withConfig(t)
	repo := setupTestRepo(t)

	initCmd := &InitCommand{}
	if err := initCmd.execute(repo); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	setupScript := filepath.Join(repo, ".conductor-cli", "setup")
	os.WriteFile(setupScript, []byte("#!/bin/sh\nexit 1\n"), 0755)

	createCmd := &CreateCommand{}
	wtPath, err := createCmd.execute(repo, "fail-setup")
	// Should NOT return an error — worktree was created, setup failure is a warning
	if err != nil {
		t.Fatalf("create should succeed even if setup fails: %v", err)
	}
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Error("worktree should exist even if setup script failed")
	}
}

func TestListCommand(t *testing.T) {
	withConfig(t)
	repo := setupTestRepo(t)

	initCmd := &InitCommand{}
	if err := initCmd.execute(repo); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	createCmd := &CreateCommand{}
	if _, err := createCmd.execute(repo, "feat-1"); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	var buf bytes.Buffer
	listCmd := &ListCommand{}
	if err := listCmd.execute(repo, &buf); err != nil {
		t.Fatalf("list failed: %v", err)
	}

	if !strings.Contains(buf.String(), "feat-1") {
		t.Errorf("list output missing 'feat-1': %s", buf.String())
	}
}

func TestListCommandNotRegistered(t *testing.T) {
	withConfig(t)
	repo := setupTestRepo(t)

	var buf bytes.Buffer
	listCmd := &ListCommand{}
	err := listCmd.execute(repo, &buf)
	if err == nil {
		t.Error("expected error for unregistered repo")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListAllCommand(t *testing.T) {
	withConfig(t)
	repo := setupTestRepo(t)

	initCmd := &InitCommand{}
	if err := initCmd.execute(repo); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	var buf bytes.Buffer
	listCmd := &ListCommand{All: true}
	if err := listCmd.executeAll(&buf); err != nil {
		t.Fatalf("list --all failed: %v", err)
	}

	repoName := filepath.Base(repo)
	if !strings.Contains(buf.String(), repoName) {
		t.Errorf("list --all output missing repo name %q: %s", repoName, buf.String())
	}
}

func TestListAllNoRepos(t *testing.T) {
	withConfig(t)

	var buf bytes.Buffer
	listCmd := &ListCommand{All: true}
	if err := listCmd.executeAll(&buf); err != nil {
		t.Fatalf("list --all failed: %v", err)
	}

	if !strings.Contains(buf.String(), "no repositories registered") {
		t.Errorf("expected 'no repositories registered' message, got: %s", buf.String())
	}
}

func TestListAllDeletedRepo(t *testing.T) {
	cfgPath := withConfig(t)
	repo := setupTestRepo(t)

	initCmd := &InitCommand{}
	if err := initCmd.execute(repo); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Delete the repo directory
	os.RemoveAll(repo)

	var buf bytes.Buffer
	listCmd := &ListCommand{All: true}
	if err := listCmd.executeAll(&buf); err != nil {
		t.Fatalf("list --all should not fail for deleted repo: %v", err)
	}

	// Config should still have the repo
	cfg, _ := config.Load(cfgPath)
	if len(cfg.Repositories) != 1 {
		t.Error("deleted repo should still be in config")
	}
}

func TestArchiveCommand(t *testing.T) {
	withConfig(t)
	repo := setupTestRepo(t)

	initCmd := &InitCommand{}
	if err := initCmd.execute(repo); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	createCmd := &CreateCommand{}
	wtPath, err := createCmd.execute(repo, "to-archive")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Fatal("worktree should exist before archive")
	}

	archiveCmd := &ArchiveCommand{}
	if err := archiveCmd.execute(repo, "to-archive"); err != nil {
		t.Fatalf("archive failed: %v", err)
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Error("worktree directory should be removed after archive")
	}

	// Branch should still exist
	cmd := exec.Command("git", "branch", "--list", "to-archive")
	cmd.Dir = repo
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git branch --list failed: %v", err)
	}
	if !strings.Contains(string(out), "to-archive") {
		t.Error("branch should still exist after archive")
	}
}

func TestArchiveCommandNotFound(t *testing.T) {
	withConfig(t)
	repo := setupTestRepo(t)

	initCmd := &InitCommand{}
	if err := initCmd.execute(repo); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	archiveCmd := &ArchiveCommand{}
	err := archiveCmd.execute(repo, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent branch")
	}
	if !strings.Contains(err.Error(), "no worktree found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestArchiveCommandFromInsideTarget(t *testing.T) {
	withConfig(t)
	repo := setupTestRepo(t)

	initCmd := &InitCommand{}
	if err := initCmd.execute(repo); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	createCmd := &CreateCommand{}
	wtPath, err := createCmd.execute(repo, "inside-test")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	archiveCmd := &ArchiveCommand{}
	err = archiveCmd.execute(wtPath, "inside-test")
	if err == nil {
		t.Error("expected error when archiving from inside target worktree")
	}
	if !strings.Contains(err.Error(), "cannot archive") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestArchiveCommandDeletedDirectory(t *testing.T) {
	withConfig(t)
	repo := setupTestRepo(t)

	initCmd := &InitCommand{}
	if err := initCmd.execute(repo); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	createCmd := &CreateCommand{}
	wtPath, err := createCmd.execute(repo, "deleted-wt")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Manually delete the worktree directory
	os.RemoveAll(wtPath)

	archiveCmd := &ArchiveCommand{}
	err = archiveCmd.execute(repo, "deleted-wt")
	if err != nil {
		t.Fatalf("archive of deleted worktree should succeed: %v", err)
	}
}
