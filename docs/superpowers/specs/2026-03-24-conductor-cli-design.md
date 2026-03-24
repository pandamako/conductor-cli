# conductor-cli Design Spec

CLI tool for efficient git worktree management across projects.

## Overview

`conductor-cli` manages git worktrees by placing them outside the project directory in a centralized location. It maintains a global registry of repositories and delegates worktree tracking to git itself.

**Stack:** Go 1.26, go-flags

## Data Model

### Global Configuration

Location: `~/.conductor-cli/config.json`

```json
{
  "worktrees_base_path": "~/.conductor-cli/worktrees",
  "repositories": {
    "/Users/user/Projects/my-app": {
      "name": "my-app"
    }
  }
}
```

- `worktrees_base_path` — base directory for all worktrees, user-overridable. Default: `~/.conductor-cli/worktrees`.
- `repositories` — map keyed by absolute path to git root. `name` is derived from the last segment of the directory path at init time.

### Project Configuration

Location: `<repo>/.conductor-cli/setup`

An optional executable file. Created by the user manually. Executed after worktree creation with working directory set to the new worktree path.

### Worktree Directory Layout

```
~/.conductor-cli/worktrees/<repo-name>/<branch-name>/
```

## Commands

### `conductor-cli init`

Registers the current git repository with conductor-cli.

- Must be executed from within a git repository
- Determines git root via `git rev-parse --show-toplevel`
- Extracts repository name from the last path segment
- Adds entry to `~/.conductor-cli/config.json`
- Creates `<repo>/.conductor-cli/` directory if it does not exist

**Errors:**
- Not a git repository — error: "not a git repository"
- Already registered — message: "repository already registered"

### `conductor-cli create <branch-name>`

Creates a new worktree with a new branch from current HEAD.

- Must be executed from within a registered repository
- Runs `git worktree add <worktrees_base_path>/<repo-name>/<branch-name> -b <branch-name>`
- If `<repo>/.conductor-cli/setup` exists and is executable — runs it with working directory set to the created worktree
- Prints the path to the created worktree

**Errors:**
- Repository not registered — error: "repository not initialized, run conductor-cli init"
- Branch already exists — git error, passed through
- Worktree directory already exists — git error, passed through
- Setup script exists but not executable — error with hint: `chmod +x .conductor-cli/setup`
- Setup script fails — print stderr, worktree remains (already created)
- Setup script not found — silently skip (script is optional)

### `conductor-cli list`

Lists worktrees for the current repository.

- Must be executed from within a registered repository
- Calls `git worktree list` and displays formatted output

**Errors:**
- Repository not registered — error: "repository not initialized, run conductor-cli init"

### `conductor-cli list --all`

Lists all registered repositories and their worktrees.

- Can be executed from any directory
- Iterates over all repositories in `config.json`
- For each repository, calls `git worktree list` and displays results grouped by repository name

**Errors:**
- No registered repositories — message: "no repositories registered"
- Repository directory no longer exists on disk — skip with warning

### `conductor-cli archive <branch-name>`

Removes a worktree. The branch is preserved.

- Must be executed from within a registered repository
- Finds the worktree matching the branch name via `git worktree list`
- Calls `git worktree remove <path>`
- On success — reports removal
- On failure — error is displayed, worktree remains in git's list (can retry)

**Errors:**
- Repository not registered — error: "repository not initialized, run conductor-cli init"
- No worktree found for branch — error: "no worktree found for branch X"
- Worktree directory already deleted — run `git worktree prune`, then report

## Code Structure

```
conductor-cli/
  cmd/conductor-cli/
    main.go          # entry point, go-flags command registration
    init.go          # InitCommand struct + Execute
    create.go        # CreateCommand struct + Execute
    list.go          # ListCommand struct + Execute (--all flag)
    archive.go       # ArchiveCommand struct + Execute
  internal/
    config/
      config.go      # GlobalConfig struct, Load/Save, path helpers
    git/
      git.go         # wrappers: WorktreeAdd, WorktreeRemove, WorktreeList, RevParseRoot
  go.mod
  go.sum
```

## Dependencies

- `github.com/jessevdk/go-flags` — CLI argument parsing
- Go standard library: `os/exec`, `encoding/json`, `os`, `path/filepath`

Git interaction is done via `os/exec` calls to the `git` CLI. No go-git or similar libraries.

## Design Decisions

1. **Git is the source of truth for worktree list.** conductor-cli does not maintain its own worktree registry. `git worktree list` is authoritative.
2. **Global config only stores repository registry and base path.** Per-project configuration (setup script) lives in the repository itself under `.conductor-cli/`.
3. **Setup script is optional.** Its absence is not an error.
4. **Archive preserves branches.** Only the worktree is removed.
5. **Errors from git are passed through.** conductor-cli does not wrap or reinterpret git errors for branch/worktree conflicts.
6. **Failed archive does not corrupt state.** If `git worktree remove` fails, the worktree stays in git's list and can be retried.
