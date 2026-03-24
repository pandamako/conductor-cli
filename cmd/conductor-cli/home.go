package main

import (
	"fmt"
	"os"

	"conductor-cli/internal/git"
)

type HomeCommand struct{}

func (c *HomeCommand) Execute(args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	path, err := c.execute(cwd)
	if err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

func (c *HomeCommand) execute(dir string) (string, error) {
	repoRoot, err := git.ResolveMainWorktree(dir)
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}
	return repoRoot, nil
}
