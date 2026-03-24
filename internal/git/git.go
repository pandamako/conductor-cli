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

func BranchExists(dir, branch string) bool {
	_, err := run(dir, "rev-parse", "--verify", "refs/heads/"+branch)
	return err == nil
}

func WorktreeAdd(dir, path, branch string) error {
	if BranchExists(dir, branch) {
		_, err := run(dir, "worktree", "add", path, branch)
		return err
	}
	_, err := run(dir, "worktree", "add", path, "-b", branch)
	return err
}

func WorktreeRemove(dir, path string) error {
	_, err := run(dir, "worktree", "remove", "--force", path)
	return err
}

func WorktreePrune(dir string) error {
	_, err := run(dir, "worktree", "prune")
	return err
}
