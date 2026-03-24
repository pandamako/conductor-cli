# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
go build ./cmd/conductor-cli/          # Build binary (outputs ./conductor-cli)
go test ./...                          # Run all tests
go test ./... -count=1                 # Run all tests (skip cache)
go test ./cmd/conductor-cli/ -v -run TestInit  # Run specific test group
go test ./internal/config/ -v          # Run package tests
```

## Architecture

CLI tool for managing git worktrees across projects. Three-layer structure:

- **`cmd/conductor-cli/`** — Thin command handlers. Each command has `Execute()` (called by go-flags) and `execute(dir)` (testable, takes explicit working directory). Commands are registered in `main.go` via `parser.AddCommand()`.
- **`internal/config/`** — Global config (`~/.conductor-cli/config.json`) load/save, branch name sanitization (slashes become nested subdirectories), path helpers. Stateless functions, no global state.
- **`internal/git/`** — Git CLI wrappers via `os/exec`. Parses `git worktree list --porcelain` output. Key functions: `ResolveMainWorktree` (works from any worktree via `--git-common-dir`), `RevParseRoot` (main tree only via `--show-toplevel`).

**Config path:** Package-level `configPath` variable in `cmd/conductor-cli/main.go`. Defaults to `~/.conductor-cli/config.json`, overridable via `CONDUCTOR_CONFIG` env var. Tests override it directly via `withConfig()` helper.

**Git is source of truth** for worktree list — conductor-cli does not maintain its own worktree registry.

## Key Patterns

- **go-flags:** Commands are structs with tagged fields for flags. `go-flags` calls `Execute(args []string) error` on the active command.
- **macOS symlinks:** `ResolveMainWorktree` uses `filepath.EvalSymlinks` to normalize `/var` vs `/private/var`. Tests use `resolveSymlinks()` helper for assertions.
- **Setup scripts:** `<repo>/.conductor-cli/setup` — optional, checked for executability *before* worktree creation to avoid orphaned worktrees.
- **Test isolation:** `withConfig(t)` pre-seeds a temp config with temp `WorktreesBasePath` so tests never touch `~/.conductor-cli/`. Tests must NOT use `t.Parallel()` due to shared `configPath`.
