# conductor-cli Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a CLI tool for efficient git worktree management across projects.

**Architecture:** Flat Go project with `cmd/conductor-cli/` for CLI entry point and commands, `internal/config/` for global config management, `internal/git/` for git CLI wrappers. Commands are thin wrappers that wire internal packages together.

**Tech Stack:** Go 1.22+, github.com/jessevdk/go-flags

**Spec:** `docs/superpowers/specs/2026-03-24-conductor-cli-design.md`

---

## File Structure

```
conductor-cli/
  cmd/conductor-cli/
    main.go              # entry point, go-flags parser, configPath variable
    init.go              # InitCommand struct + Execute
    create.go            # CreateCommand struct + Execute
    list.go              # ListCommand struct + Execute (--all flag)
    archive.go           # ArchiveCommand struct + Execute
    commands_test.go     # integration tests for all commands
  internal/
    config/
      config.go          # GlobalConfig struct, Load/Save, SanitizeBranchName, path helpers
      config_test.go     # unit tests for config package
    git/
      git.go             # CLI wrappers: WorktreeAdd/Remove/List, RevParseRoot, ResolveMainWorktree
      git_test.go        # unit tests for git package (porcelain parsing + real git repos)
  go.mod
  go.sum
```

**Config path resolution:** A package-level `configPath` variable in `cmd/conductor-cli/` defaults to `DefaultConfigPath()` but checks `CONDUCTOR_CONFIG` env var first. Tests override `configPath` directly. No `--config` CLI flag in v1.

---

### Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `cmd/conductor-cli/main.go`

- [ ] **Step 1: Initialize Go module and install dependency**

Run:
```bash
cd /Users/artembykov/Documents/Projects/conductor-cli
go mod init conductor-cli
go get github.com/jessevdk/go-flags
```

- [ ] **Step 2: Create main.go**

Create `cmd/conductor-cli/main.go`:

```go
package main

import (
	"os"

	"conductor-cli/internal/config"

	"github.com/jessevdk/go-flags"
)

// configPath is the path to the global config file.
// Defaults to ~/.conductor-cli/config.json, overridable via CONDUCTOR_CONFIG env var.
// Tests override this directly.
var configPath = func() string {
	if v := os.Getenv("CONDUCTOR_CONFIG"); v != "" {
		return v
	}
	return config.DefaultConfigPath()
}()

func main() {
	parser := flags.NewParser(nil, flags.Default)

	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Create stub config package so main.go compiles**

Create `internal/config/config.go`:

```go
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
```

- [ ] **Step 4: Verify it compiles**

Run: `go build ./cmd/conductor-cli/`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum cmd/ internal/
git commit -m "feat: project scaffolding with go-flags"
```

---

### Task 2: Config — Branch Name Sanitization

**Files:**
- Modify: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests for SanitizeBranchName and UnsanitizeBranchName**

Create `internal/config/config_test.go`:

```go
package config

import "testing"

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"main", "main"},
		{"feature/login", "feature%2Flogin"},
		{"feature/deep/nested", "feature%2Fdeep%2Fnested"},
		{"has%percent", "has%25percent"},
		{"feature%2Flogin", "feature%252Flogin"},
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

func TestUnsanitizeBranchName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"main", "main"},
		{"feature%2Flogin", "feature/login"},
		{"feature%2Fdeep%2Fnested", "feature/deep/nested"},
		{"has%25percent", "has%percent"},
		{"feature%252Flogin", "feature%2Flogin"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := UnsanitizeBranchName(tt.input)
			if got != tt.expected {
				t.Errorf("UnsanitizeBranchName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSanitizeRoundTrip(t *testing.T) {
	names := []string{"main", "feature/login", "a%b/c%2Fd", "deep/a/b/c"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			got := UnsanitizeBranchName(SanitizeBranchName(name))
			if got != name {
				t.Errorf("round-trip failed for %q: got %q", name, got)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -v`
Expected: FAIL — SanitizeBranchName, UnsanitizeBranchName not defined

- [ ] **Step 3: Implement SanitizeBranchName and UnsanitizeBranchName**

Add to `internal/config/config.go` (after existing code):

```go
import "strings"

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
```

Note: merge the `"strings"` import into the existing import block.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: branch name sanitization for directory names"
```

---

### Task 3: Config — Load, Save, and Path Helpers

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests**

Add the following test functions to `internal/config/config_test.go`. Merge new imports (`"os"`, `"path/filepath"`) into the existing import block.

```go
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
	expected := filepath.Join("/base", "myrepo", "feature%2Flogin")
	if got != expected {
		t.Errorf("WorktreePath = %q, want %q", got, expected)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -v`
Expected: FAIL — GlobalConfig, Load, Save, RepoConfig not defined

- [ ] **Step 3: Implement GlobalConfig, Load, Save, helpers**

Replace `internal/config/config.go` with the complete file (merging Task 1 stub + Task 2 sanitization + new code):

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: config load/save with tilde expansion and path helpers"
```

---

### Task 4: Git — Porcelain Output Parsing

**Files:**
- Create: `internal/git/git.go`
- Create: `internal/git/git_test.go`

- [ ] **Step 1: Write failing tests for ParsePorcelainOutput**

Create `internal/git/git_test.go`:

```go
package git

import (
	"strings"
	"testing"
)

func TestParsePorcelainOutput(t *testing.T) {
	input := strings.Join([]string{
		"worktree /home/user/repo",
		"HEAD abc123def456",
		"branch refs/heads/main",
		"",
		"worktree /home/user/worktrees/feature",
		"HEAD def789abc012",
		"branch refs/heads/feature/login",
		"",
		"worktree /home/user/worktrees/detached",
		"HEAD 111222333444",
		"detached",
		"",
	}, "\n")

	worktrees := ParsePorcelainOutput(input)

	if len(worktrees) != 3 {
		t.Fatalf("expected 3 worktrees, got %d", len(worktrees))
	}

	if worktrees[0].Path != "/home/user/repo" {
		t.Errorf("wt[0].Path = %q", worktrees[0].Path)
	}
	if worktrees[0].Branch != "main" {
		t.Errorf("wt[0].Branch = %q", worktrees[0].Branch)
	}
	if worktrees[0].Head != "abc123def456" {
		t.Errorf("wt[0].Head = %q", worktrees[0].Head)
	}

	if worktrees[1].Branch != "feature/login" {
		t.Errorf("wt[1].Branch = %q, want %q", worktrees[1].Branch, "feature/login")
	}

	if worktrees[2].Branch != "" {
		t.Errorf("wt[2].Branch = %q, want empty", worktrees[2].Branch)
	}
	if !worktrees[2].Detached {
		t.Error("wt[2] should be detached")
	}
}

func TestParsePorcelainOutputEmpty(t *testing.T) {
	worktrees := ParsePorcelainOutput("")
	if len(worktrees) != 0 {
		t.Errorf("expected 0 worktrees, got %d", len(worktrees))
	}
}

func TestParsePorcelainOutputBare(t *testing.T) {
	input := "worktree /home/user/bare-repo\nHEAD abc123\nbare\n\n"
	worktrees := ParsePorcelainOutput(input)
	if len(worktrees) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(worktrees))
	}
	if !worktrees[0].Bare {
		t.Error("expected bare worktree")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/git/ -v`
Expected: FAIL — types and functions not defined

- [ ] **Step 3: Implement Worktree struct and ParsePorcelainOutput**

Create `internal/git/git.go`:

```go
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type Worktree struct {
	Path     string
	Head     string
	Branch   string
	Bare     bool
	Detached bool
}

func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

// ParsePorcelainOutput parses the output of `git worktree list --porcelain`.
func ParsePorcelainOutput(output string) []Worktree {
	if output == "" {
		return nil
	}
	var worktrees []Worktree
	var current Worktree
	hasEntry := false

	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			if hasEntry {
				worktrees = append(worktrees, current)
			}
			current = Worktree{Path: strings.TrimPrefix(line, "worktree ")}
			hasEntry = true
		case strings.HasPrefix(line, "HEAD "):
			current.Head = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			branch := strings.TrimPrefix(line, "branch ")
			current.Branch = strings.TrimPrefix(branch, "refs/heads/")
		case line == "bare":
			current.Bare = true
		case line == "detached":
			current.Detached = true
		}
	}
	if hasEntry {
		worktrees = append(worktrees, current)
	}
	return worktrees
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/git/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/git/
git commit -m "feat: git worktree porcelain output parsing"
```

---

### Task 5: Git — CLI Wrappers

**Files:**
- Modify: `internal/git/git.go`
- Modify: `internal/git/git_test.go`

- [ ] **Step 1: Write failing tests for RevParseRoot and ResolveMainWorktree**

Add the following to `internal/git/git_test.go`. Merge new imports (`"os"`, `"os/exec"`, `"path/filepath"`) into the existing import block.

```go
// setupGitRepo creates a temp git repo with an initial commit.
func setupGitRepo(t *testing.T) string {
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

// resolveSymlinks resolves macOS /private/var vs /var differences.
func resolveSymlinks(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return resolved
}

func TestRevParseRoot(t *testing.T) {
	repo := setupGitRepo(t)
	subdir := filepath.Join(repo, "sub", "dir")
	os.MkdirAll(subdir, 0755)

	root, err := RevParseRoot(subdir)
	if err != nil {
		t.Fatalf("RevParseRoot() error = %v", err)
	}

	if resolveSymlinks(root) != resolveSymlinks(repo) {
		t.Errorf("RevParseRoot() = %q, want %q", root, repo)
	}
}

func TestRevParseRootNotGit(t *testing.T) {
	dir := t.TempDir()
	_, err := RevParseRoot(dir)
	if err == nil {
		t.Error("expected error for non-git directory")
	}
}

func TestResolveMainWorktreeFromMain(t *testing.T) {
	repo := setupGitRepo(t)

	resolved, err := ResolveMainWorktree(repo)
	if err != nil {
		t.Fatalf("ResolveMainWorktree() error = %v", err)
	}

	if resolveSymlinks(resolved) != resolveSymlinks(repo) {
		t.Errorf("ResolveMainWorktree() = %q, want %q", resolved, repo)
	}
}

func TestResolveMainWorktreeFromLinked(t *testing.T) {
	repo := setupGitRepo(t)
	wtPath := filepath.Join(t.TempDir(), "linked-wt")

	cmd := exec.Command("git", "worktree", "add", wtPath, "-b", "test-branch")
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add failed: %s", out)
	}

	resolved, err := ResolveMainWorktree(wtPath)
	if err != nil {
		t.Fatalf("ResolveMainWorktree() error = %v", err)
	}

	if resolveSymlinks(resolved) != resolveSymlinks(repo) {
		t.Errorf("ResolveMainWorktree() from linked = %q, want %q", resolved, repo)
	}
}

func TestWorktreeListIntegration(t *testing.T) {
	repo := setupGitRepo(t)
	wtPath := filepath.Join(t.TempDir(), "wt")

	cmd := exec.Command("git", "worktree", "add", wtPath, "-b", "feat")
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add failed: %s", out)
	}

	worktrees, err := WorktreeList(repo)
	if err != nil {
		t.Fatalf("WorktreeList() error = %v", err)
	}
	if len(worktrees) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(worktrees))
	}

	found := false
	for _, wt := range worktrees {
		if wt.Branch == "feat" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'feat' branch in worktree list")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/git/ -v -run "TestRevParse|TestResolve|TestWorktreeList"`
Expected: FAIL — functions not defined

- [ ] **Step 3: Implement CLI wrappers**

Add to `internal/git/git.go`:

```go
func RevParseRoot(dir string) (string, error) {
	return run(dir, "rev-parse", "--show-toplevel")
}

func ResolveMainWorktree(dir string) (string, error) {
	gitCommonDir, err := run(dir, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(gitCommonDir) {
		gitCommonDir = filepath.Join(dir, gitCommonDir)
	}
	absPath, err := filepath.Abs(gitCommonDir)
	if err != nil {
		return "", err
	}
	// EvalSymlinks to match git rev-parse --show-toplevel behavior on macOS
	// (where /var is a symlink to /private/var)
	resolved, err := filepath.EvalSymlinks(filepath.Dir(absPath))
	if err != nil {
		return filepath.Dir(absPath), nil
	}
	return resolved, nil
}

func WorktreeList(dir string) ([]Worktree, error) {
	output, err := run(dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return ParsePorcelainOutput(output), nil
}

func WorktreeAdd(dir, path, branch string) error {
	_, err := run(dir, "worktree", "add", path, "-b", branch)
	return err
}

func WorktreeRemove(dir, path string) error {
	_, err := run(dir, "worktree", "remove", path)
	return err
}

func WorktreePrune(dir string) error {
	_, err := run(dir, "worktree", "prune")
	return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/git/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/git/
git commit -m "feat: git CLI wrappers for worktree operations"
```

---

### Task 6: Init Command

**Files:**
- Create: `cmd/conductor-cli/init.go`
- Create: `cmd/conductor-cli/commands_test.go`
- Modify: `cmd/conductor-cli/main.go` (register command)

- [ ] **Step 1: Write failing tests for init**

Create `cmd/conductor-cli/commands_test.go`:

```go
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/conductor-cli/ -v -run TestInit`
Expected: FAIL — InitCommand not defined

- [ ] **Step 3: Implement init command**

Create `cmd/conductor-cli/init.go`:

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"conductor-cli/internal/config"
	"conductor-cli/internal/git"
)

type InitCommand struct {
	Name string `long:"name" description:"Custom repository name"`
}

func (c *InitCommand) Execute(args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return c.execute(cwd)
}

func (c *InitCommand) execute(dir string) error {
	repoRoot, err := git.RevParseRoot(dir)
	if err != nil {
		return fmt.Errorf("not a git repository")
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if _, exists := cfg.Repositories[repoRoot]; exists {
		fmt.Println("repository already registered")
		return nil
	}

	name := c.Name
	if name == "" {
		name = filepath.Base(repoRoot)
	}
	if name == "" {
		return fmt.Errorf("could not derive repository name")
	}

	if existingPath, found := cfg.FindRepoByName(name); found {
		return fmt.Errorf("name '%s' already used by %s, use --name to specify a different name", name, existingPath)
	}

	cfg.Repositories[repoRoot] = config.RepoConfig{Name: name}

	if err := config.Save(configPath, cfg); err != nil {
		return err
	}

	conductorDir := filepath.Join(repoRoot, ".conductor-cli")
	if err := os.MkdirAll(conductorDir, 0755); err != nil {
		return err
	}

	fmt.Printf("repository '%s' registered\n", name)
	return nil
}
```

Register command in `cmd/conductor-cli/main.go` — add after `parser := flags.NewParser(...)`:

```go
parser.AddCommand("init", "Register repository", "Register the current git repository with conductor-cli", &InitCommand{})
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/conductor-cli/ -v -run TestInit`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/conductor-cli/
git commit -m "feat: init command — register repository"
```

---

### Task 7: Create Command

**Files:**
- Create: `cmd/conductor-cli/create.go`
- Modify: `cmd/conductor-cli/commands_test.go` (add tests)
- Modify: `cmd/conductor-cli/main.go` (register command)

- [ ] **Step 1: Write failing tests for create**

Add the following test functions to `cmd/conductor-cli/commands_test.go`. Add `"conductor-cli/internal/git"` to the import block.

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/conductor-cli/ -v -run TestCreate`
Expected: FAIL — CreateCommand not defined

- [ ] **Step 3: Implement create command**

Create `cmd/conductor-cli/create.go`:

```go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"conductor-cli/internal/config"
	"conductor-cli/internal/git"
)

type CreateCommand struct{}

func (c *CreateCommand) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: conductor-cli create <branch-name>")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	wtPath, err := c.execute(cwd, args[0])
	if err != nil {
		return err
	}
	fmt.Println(wtPath)
	return nil
}

func (c *CreateCommand) execute(dir, branchName string) (string, error) {
	repoRoot, err := git.ResolveMainWorktree(dir)
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return "", err
	}

	repo, exists := cfg.Repositories[repoRoot]
	if !exists {
		return "", fmt.Errorf("repository not initialized, run conductor-cli init")
	}

	wtPath := cfg.WorktreePath(repo.Name, branchName)

	if err := git.WorktreeAdd(dir, wtPath, branchName); err != nil {
		return "", err
	}

	// Run setup script if it exists
	setupScript := filepath.Join(repoRoot, ".conductor-cli", "setup")
	info, statErr := os.Stat(setupScript)
	if statErr == nil {
		if info.Mode()&0111 == 0 {
			return "", fmt.Errorf("setup script is not executable, run: chmod +x .conductor-cli/setup")
		}
		cmd := exec.Command(setupScript)
		cmd.Dir = wtPath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: setup script failed: %v\n", err)
		}
	}

	return wtPath, nil
}
```

Register in `cmd/conductor-cli/main.go`:

```go
parser.AddCommand("create", "Create worktree", "Create a new worktree with a new branch", &CreateCommand{})
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/conductor-cli/ -v -run TestCreate`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/conductor-cli/
git commit -m "feat: create command — worktree creation with setup script"
```

---

### Task 8: List Command

**Files:**
- Create: `cmd/conductor-cli/list.go`
- Modify: `cmd/conductor-cli/commands_test.go` (add tests)
- Modify: `cmd/conductor-cli/main.go` (register command)

- [ ] **Step 1: Write failing tests for list**

Add the following test functions to `cmd/conductor-cli/commands_test.go`. Add `"bytes"` to the import block.

```go
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
	// Should not error — just skip with warning
	if err := listCmd.executeAll(&buf); err != nil {
		t.Fatalf("list --all should not fail for deleted repo: %v", err)
	}

	// Verify config still has the repo (not removed)
	cfg, _ := config.Load(cfgPath)
	if len(cfg.Repositories) != 1 {
		t.Error("deleted repo should still be in config")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/conductor-cli/ -v -run TestList`
Expected: FAIL — ListCommand not defined

- [ ] **Step 3: Implement list command**

Create `cmd/conductor-cli/list.go`:

```go
package main

import (
	"fmt"
	"io"
	"os"

	"conductor-cli/internal/config"
	"conductor-cli/internal/git"
)

type ListCommand struct {
	All bool `long:"all" description:"List worktrees for all registered repositories"`
}

func (c *ListCommand) Execute(args []string) error {
	if c.All {
		return c.executeAll(os.Stdout)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return c.execute(cwd, os.Stdout)
}

func (c *ListCommand) execute(dir string, w io.Writer) error {
	repoRoot, err := git.ResolveMainWorktree(dir)
	if err != nil {
		return fmt.Errorf("not a git repository")
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if _, exists := cfg.Repositories[repoRoot]; !exists {
		return fmt.Errorf("repository not initialized, run conductor-cli init")
	}

	worktrees, err := git.WorktreeList(repoRoot)
	if err != nil {
		return err
	}

	for _, wt := range worktrees {
		branch := wt.Branch
		if branch == "" {
			branch = "(detached)"
		}
		fmt.Fprintf(w, "%-40s %s\n", branch, wt.Path)
	}
	return nil
}

func (c *ListCommand) executeAll(w io.Writer) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if len(cfg.Repositories) == 0 {
		fmt.Fprintln(w, "no repositories registered")
		return nil
	}

	for repoPath, repo := range cfg.Repositories {
		if _, err := os.Stat(repoPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "warning: %s (%s) no longer exists, skipping\n", repo.Name, repoPath)
			continue
		}

		worktrees, err := git.WorktreeList(repoPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to list worktrees for %s: %v\n", repo.Name, err)
			continue
		}

		fmt.Fprintf(w, "%s (%s)\n", repo.Name, repoPath)
		for _, wt := range worktrees {
			branch := wt.Branch
			if branch == "" {
				branch = "(detached)"
			}
			fmt.Fprintf(w, "  %-38s %s\n", branch, wt.Path)
		}
		fmt.Fprintln(w)
	}
	return nil
}
```

Register in `cmd/conductor-cli/main.go`:

```go
parser.AddCommand("list", "List worktrees", "List worktrees for current or all repositories", &ListCommand{})
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/conductor-cli/ -v -run TestList`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/conductor-cli/
git commit -m "feat: list command — current repo and --all modes"
```

---

### Task 9: Archive Command

**Files:**
- Create: `cmd/conductor-cli/archive.go`
- Modify: `cmd/conductor-cli/commands_test.go` (add tests)
- Modify: `cmd/conductor-cli/main.go` (register command)

- [ ] **Step 1: Write failing tests for archive**

Add the following test functions to `cmd/conductor-cli/commands_test.go`:

```go
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

	// Manually delete the worktree directory (simulating user rm -rf)
	os.RemoveAll(wtPath)

	archiveCmd := &ArchiveCommand{}
	err = archiveCmd.execute(repo, "deleted-wt")
	// Should succeed — prune the stale entry
	if err != nil {
		t.Fatalf("archive of deleted worktree should succeed: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/conductor-cli/ -v -run TestArchive`
Expected: FAIL — ArchiveCommand not defined

- [ ] **Step 3: Implement archive command**

Create `cmd/conductor-cli/archive.go`:

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"conductor-cli/internal/config"
	"conductor-cli/internal/git"
)

type ArchiveCommand struct{}

func (c *ArchiveCommand) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: conductor-cli archive <branch-name>")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return c.execute(cwd, args[0])
}

func (c *ArchiveCommand) execute(dir, branchName string) error {
	repoRoot, err := git.ResolveMainWorktree(dir)
	if err != nil {
		return fmt.Errorf("not a git repository")
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if _, exists := cfg.Repositories[repoRoot]; !exists {
		return fmt.Errorf("repository not initialized, run conductor-cli init")
	}

	worktrees, err := git.WorktreeList(repoRoot)
	if err != nil {
		return err
	}

	var targetPath string
	for _, wt := range worktrees {
		if wt.Branch == branchName {
			targetPath = wt.Path
			break
		}
	}

	if targetPath == "" {
		return fmt.Errorf("no worktree found for branch '%s'", branchName)
	}

	// Check if user is inside the target worktree.
	// Use EvalSymlinks to handle macOS /private/var vs /var.
	absDir, _ := filepath.Abs(dir)
	absTarget, _ := filepath.Abs(targetPath)
	absDir = resolveLinks(absDir)
	absTarget = resolveLinks(absTarget)
	if absDir == absTarget || strings.HasPrefix(absDir, absTarget+string(filepath.Separator)) {
		return fmt.Errorf("cannot archive worktree you are currently in, cd to another directory first")
	}

	// Check if directory still exists
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		if err := git.WorktreePrune(repoRoot); err != nil {
			return err
		}
		fmt.Printf("worktree for branch '%s' pruned (directory was already removed)\n", branchName)
		return nil
	}

	if err := git.WorktreeRemove(repoRoot, targetPath); err != nil {
		return err
	}

	fmt.Printf("worktree for branch '%s' archived\n", branchName)
	return nil
}

func resolveLinks(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return resolved
}
```

Register in `cmd/conductor-cli/main.go`:

```go
parser.AddCommand("archive", "Archive worktree", "Remove a worktree (branch is preserved)", &ArchiveCommand{})
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/conductor-cli/ -v -run TestArchive`
Expected: PASS

- [ ] **Step 5: Run all tests**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
git add cmd/conductor-cli/
git commit -m "feat: archive command — remove worktree, preserve branch"
```
