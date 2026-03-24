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

	parser.AddCommand("init", "Register repository", "Register the current git repository with conductor-cli", &InitCommand{})

	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}
}
