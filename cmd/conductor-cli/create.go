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

	// Check setup script executability before creating worktree
	setupScript := filepath.Join(repoRoot, ".conductor-cli", "setup")
	info, statErr := os.Stat(setupScript)
	if statErr == nil && info.Mode()&0111 == 0 {
		return "", fmt.Errorf("setup script is not executable, run: chmod +x .conductor-cli/setup")
	}

	if err := git.WorktreeAdd(dir, wtPath, branchName); err != nil {
		return "", err
	}

	// Run setup script if it exists
	if statErr == nil {
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
