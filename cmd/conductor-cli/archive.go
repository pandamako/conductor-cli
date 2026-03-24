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
