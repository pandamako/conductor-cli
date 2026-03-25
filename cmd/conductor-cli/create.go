package main

import (
	"fmt"
	"io"
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

	// Copy .conductor-cli/ from main repo to worktree (may contain gitignored files)
	conductorDir := filepath.Join(repoRoot, ".conductor-cli")
	if _, err := os.Stat(conductorDir); err == nil {
		if err := copyDir(conductorDir, filepath.Join(wtPath, ".conductor-cli")); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to copy .conductor-cli: %v\n", err)
		}
	}

	// Run setup script if it exists
	wtSetupScript := filepath.Join(wtPath, ".conductor-cli", "setup")
	if info, err := os.Stat(wtSetupScript); err == nil && info.Mode()&0111 != 0 {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "sh"
		}
		cmd := exec.Command(shell, "-li", wtSetupScript)
		cmd.Dir = wtPath
		cmd.Env = append(os.Environ(), "CONDUCTOR_MAIN_WORKTREE="+repoRoot)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: setup script failed: %v\n", err)
		}
	}

	return wtPath, nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
