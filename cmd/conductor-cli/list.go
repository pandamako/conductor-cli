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
