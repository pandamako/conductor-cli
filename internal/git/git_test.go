package git

import (
	"os"
	"os/exec"
	"path/filepath"
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
