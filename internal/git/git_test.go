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
