# conductor-cli

CLI tool for managing git worktrees across projects.

## Installation

### Homebrew (macOS/Linux)

```bash
brew install pandamako/tap/conductor-cli
```

### From source

```bash
go install github.com/pandamako/conductor-cli/cmd/conductor-cli@latest
```

### Binary releases

Download pre-built binaries from the [Releases](https://github.com/pandamako/conductor-cli/releases) page.

## Commands

### `init`

Register the current git repository with conductor-cli.

```bash
conductor-cli init [--name <custom-name>]
```

Creates a `.conductor-cli/` directory in the repository root with a `setup` script template. The repository name defaults to the directory name; use `--name` to override.

### `create`

Create a new worktree with a new branch.

```bash
conductor-cli create <branch-name>
```

Creates a git worktree in the configured base path. Copies `.conductor-cli/` directory to the new worktree and runs the `setup` script if it exists and is executable. Prints the path to the created worktree.

### `list`

List worktrees for the current or all registered repositories.

```bash
conductor-cli list [--all]
```

Without flags, lists worktrees for the current repository. With `--all`, lists worktrees for every registered repository.

### `archive`

Remove a worktree. The git branch is preserved.

```bash
conductor-cli archive <branch-name>
```

Removes the worktree directory but keeps the branch in git. Cannot archive the worktree you are currently in.

### `home`

Print the path to the main worktree of the current repository.

```bash
conductor-cli home
```

Works from any worktree of the repository.

## Setup Script

After `init`, a template script is created at `.conductor-cli/setup`. Edit it to define commands that run automatically after each `create`. The script receives the `CONDUCTOR_MAIN_WORKTREE` environment variable with the absolute path to the main worktree.

## Configuration

Global config is stored at `~/.conductor-cli/config.json`. Override with the `CONDUCTOR_CONFIG` environment variable.
