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

	// Create setup script template if it doesn't exist
	setupScript := filepath.Join(conductorDir, "setup")
	if _, err := os.Stat(setupScript); os.IsNotExist(err) {
		template := `#!/bin/sh
# This script runs automatically after creating a new worktree.
# Working directory is set to the new worktree root.
#
# Examples:
#   npm install
#   cp .env.example .env
#   make setup
`
		if err := os.WriteFile(setupScript, []byte(template), 0755); err != nil {
			return err
		}
		fmt.Printf("Setup script created at .conductor-cli/setup\n")
		fmt.Printf("Edit it to define commands that run after worktree creation.\n")
	}

	fmt.Printf("repository '%s' registered\n", name)
	return nil
}
