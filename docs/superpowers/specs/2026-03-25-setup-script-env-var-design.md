# Design: `CONDUCTOR_MAIN_WORKTREE` Environment Variable

## Problem

Setup scripts in `.conductor-cli/setup` require absolute paths to reference files in the main worktree:

```bash
cp ~/Documents/Projects/complead/marvin/.docker_env ./
cp ~/Documents/Projects/complead/marvin/.docker_env.override ./
cp ~/Documents/Projects/complead/marvin/cmd/marvinsrv/.env.local ./cmd/marvinsrv/
```

These hardcoded paths tie the script to one machine and one user. The script cannot be committed to the repository.

## Solution

Pass the main worktree path as the environment variable `CONDUCTOR_MAIN_WORKTREE` when executing the setup script. The script becomes portable:

```bash
cp "$CONDUCTOR_MAIN_WORKTREE/.docker_env" ./
cp "$CONDUCTOR_MAIN_WORKTREE/.docker_env.override" ./
cp "$CONDUCTOR_MAIN_WORKTREE/cmd/marvinsrv/.env.local" ./cmd/marvinsrv/
```

## Changes

### `cmd/conductor-cli/create.go`

Before calling `cmd.Run()`, set the environment:

```go
cmd.Env = append(os.Environ(), "CONDUCTOR_MAIN_WORKTREE="+repoRoot)
```

`repoRoot` is already resolved at the top of `execute()` via `git.ResolveMainWorktree()`. Setting `cmd.Env` replaces the default inherited environment, so `os.Environ()` is used as the base to preserve the caller's full environment. The variable is scoped to the child process only.

### `cmd/conductor-cli/init.go`

Update the setup script template to document the variable:

```bash
#!/bin/sh
# This script runs automatically after creating a new worktree.
# Working directory is set to the new worktree root.
#
# Available environment variables:
#   CONDUCTOR_MAIN_WORKTREE — absolute path to the main worktree
#
# Examples:
#   npm install
#   cp .env.example .env
#   cp "$CONDUCTOR_MAIN_WORKTREE/.env" ./
#   make setup
```

### `cmd/conductor-cli/commands_test.go`

Add a test that verifies the variable reaches the setup script. The script writes `$CONDUCTOR_MAIN_WORKTREE` to a file; the test asserts the file contains the expected main worktree path. Path assertions must use `resolveSymlinks()` to handle macOS `/var` vs `/private/var` differences, consistent with existing tests.

## Behavior

- The variable contains an absolute, symlink-resolved path to the main worktree root.
- It exists only within the setup script's process. No global environment pollution.
- If no setup script exists, the variable is never set.
- No breaking changes. Existing setup scripts continue to work unchanged.
