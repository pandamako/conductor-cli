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

- `worktrees_base_path` — base directory for all worktrees, user-overridable. Default: `~/.conductor-cli/worktrees`. Stored as absolute path. Tilde expansion is performed at load time if the value starts with `~`.
- `repositories` — map keyed by absolute path to git root. `name` is derived from the last segment of the directory path at init time. If a name collision is detected, `init` fails with an error suggesting `conductor-cli init --name <unique-name>`.

### Project Configuration

Location: `<repo>/.conductor-cli/setup`

An optional executable file. Created by the user manually. Executed after worktree creation with working directory set to the new worktree path.

### Worktree Directory Layout

```
~/.conductor-cli/worktrees/<repo-name>/<branch-name>/
```

Slashes in branch names are URL-encoded: `feature/login` becomes `feature%2Flogin` on disk. This encoding is reversible and collision-free (a literal `%` in a branch name becomes `%25`).

## Commands

### `conductor-cli init`

Registers the current git repository with conductor-cli.

- Must be executed from within a git repository
- Determines git root via `git rev-parse --show-toplevel`
- Extracts repository name from the last path segment (overridable with `--name <name>`)
- Checks for name collision with existing repositories
- Adds entry to `~/.conductor-cli/config.json`
- Creates `<repo>/.conductor-cli/` directory if it does not exist

**Errors:**
- Not a git repository — error: "not a git repository"
- Already registered — message: "repository already registered"
- Name collision with another repository — error: "name 'X' already used by Y, use --name to specify a different name"

### `conductor-cli create <branch-name>`

Creates a new worktree with a new branch from current HEAD.

- Must be executed from within a registered repository (main working tree or an existing worktree of that repository)
- Resolves to the registered repository by finding the main worktree via `git rev-parse --git-common-dir` and matching against the registry
- Creates the branch from the HEAD of the current directory (whichever worktree the user is in)
- Runs `git worktree add <worktrees_base_path>/<repo-name>/<sanitized-branch-name> -b <branch-name>` (slashes in branch name URL-encoded for directory name)
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

- Must be executed from within a registered repository (main working tree or an existing worktree)
- Resolves to the registered repository via `git rev-parse --git-common-dir`
- Calls `git worktree list --porcelain` and displays formatted output

**Errors:**
- Repository not registered — error: "repository not initialized, run conductor-cli init"

### `conductor-cli list --all`

Lists all registered repositories and their worktrees.

- Can be executed from any directory
- Iterates over all repositories in `config.json`
- For each repository, calls `git worktree list --porcelain` and displays results grouped by repository name

**Errors:**
- No registered repositories — message: "no repositories registered"
- Repository directory no longer exists on disk — skip with warning

### `conductor-cli archive <branch-name>`

Removes a worktree. The branch is preserved.

- Must be executed from within a registered repository (main working tree or an existing worktree)
- Resolves to the registered repository via `git rev-parse --git-common-dir`
- Finds the worktree matching the branch name via `git worktree list --porcelain` (parse `branch` field for reliable matching)
- Detects if the user is currently inside the target worktree — error with hint to `cd` out first
- Calls `git worktree remove <path>`
- On success — reports removal
- On failure — error is displayed, worktree remains in git's list (can retry)

**Errors:**
- Repository not registered — error: "repository not initialized, run conductor-cli init"
- No worktree found for branch — error: "no worktree found for branch X"
- Currently inside the target worktree — error: "cannot archive worktree you are currently in, cd to another directory first"
- Worktree has uncommitted changes — git error, passed through (user can resolve and retry)
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
7. **Slashes in branch names are sanitized for directory names.** Slashes are replaced with `%2F` (URL-encoding style) to avoid ambiguity. `feature/login` becomes `feature%2Flogin` on disk. This is reversible and collision-free — a branch literally named `feature%2Flogin` would be `feature%252Flogin`. Git itself stores the real branch name; the directory name is only for filesystem layout.
8. **Repository resolution works from worktrees.** Resolution algorithm: (a) run `git rev-parse --git-common-dir`, (b) resolve to absolute path via `filepath.Abs`, (c) take parent directory (strip `/.git` suffix), (d) match against registry keys. This works both from the main working tree (where `--git-common-dir` returns `.git`) and from linked worktrees (where it returns an absolute path).
9. **Tilde expansion at load time.** `worktrees_base_path` supports `~` prefix, expanded when config is loaded.
10. **`create` only creates new branches.** Checking out existing branches as worktrees is out of scope for v1. Use `git worktree add` directly for that.
11. **`git worktree list --porcelain` for all programmatic parsing.** The human-readable format is fragile; porcelain output is stable and machine-parseable.
12. **Config file created on first `init`.** `~/.conductor-cli/` directory and `config.json` are created automatically on first `conductor-cli init` if they do not exist.
