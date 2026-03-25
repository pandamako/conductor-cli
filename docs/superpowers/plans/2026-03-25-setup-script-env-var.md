# CONDUCTOR_MAIN_WORKTREE Environment Variable — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Pass the main worktree path as `CONDUCTOR_MAIN_WORKTREE` env var to setup scripts so they can reference files portably.

**Architecture:** One line added to `create.go` sets `cmd.Env` before running the setup script. The init template is updated to document the variable. A test verifies the variable reaches the script.

**Tech Stack:** Go, os/exec, shell scripts

---

## File Map

- **Modify:** `cmd/conductor-cli/create.go:76-79` — add `cmd.Env` line
- **Modify:** `cmd/conductor-cli/init.go:66-74` — update setup script template
- **Modify:** `cmd/conductor-cli/commands_test.go` — add env var test

---

### Task 1: Add `CONDUCTOR_MAIN_WORKTREE` env var to setup script execution

**Files:**
- Modify: `cmd/conductor-cli/commands_test.go` (after line 349, before `TestListCommand`)
- Modify: `cmd/conductor-cli/create.go:76-79`

- [ ] **Step 1: Write the failing test**

Add after `TestCreateCommandSetupScriptFails` in `cmd/conductor-cli/commands_test.go`:

```go
func TestCreateCommandSetupScriptReceivesMainWorktreeEnv(t *testing.T) {
	withConfig(t)
	repo := setupTestRepo(t)

	initCmd := &InitCommand{}
	if err := initCmd.execute(repo); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	setupScript := filepath.Join(repo, ".conductor-cli", "setup")
	os.WriteFile(setupScript, []byte("#!/bin/sh\necho \"$CONDUCTOR_MAIN_WORKTREE\" > main-worktree-path\n"), 0755)

	createCmd := &CreateCommand{}
	wtPath, err := createCmd.execute(repo, "env-test")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(wtPath, "main-worktree-path"))
	if err != nil {
		t.Fatalf("setup script did not write main-worktree-path file: %v", err)
	}

	got := strings.TrimSpace(string(content))
	want := resolveSymlinks(repo)
	if got != want {
		t.Errorf("CONDUCTOR_MAIN_WORKTREE = %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/conductor-cli/ -v -run TestCreateCommandSetupScriptReceivesMainWorktreeEnv -count=1`

Expected: FAIL — `CONDUCTOR_MAIN_WORKTREE` is empty, so the file contains an empty string.

- [ ] **Step 3: Add `cmd.Env` in `create.go`**

In `cmd/conductor-cli/create.go`, add one line after `cmd.Dir = wtPath` (line 77):

```go
		cmd.Dir = wtPath
		cmd.Env = append(os.Environ(), "CONDUCTOR_MAIN_WORKTREE="+repoRoot)
		cmd.Stdout = os.Stdout
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/conductor-cli/ -v -run TestCreateCommandSetupScriptReceivesMainWorktreeEnv -count=1`

Expected: PASS

- [ ] **Step 5: Run all tests to check for regressions**

Run: `go test ./... -count=1`

Expected: All tests pass.

- [ ] **Step 6: Commit**

```bash
git add cmd/conductor-cli/create.go cmd/conductor-cli/commands_test.go
git commit -m "feat: pass CONDUCTOR_MAIN_WORKTREE env var to setup scripts"
```

---

### Task 2: Update setup script template in `init.go`

**Files:**
- Modify: `cmd/conductor-cli/init.go:66-74`

- [ ] **Step 1: Update the template**

In `cmd/conductor-cli/init.go`, replace the template string (lines 66-74) with:

```go
		template := `#!/bin/sh
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
`
```

- [ ] **Step 2: Run all tests**

Run: `go test ./... -count=1`

Expected: All tests pass (template change only affects new `init` runs).

- [ ] **Step 3: Commit**

```bash
git add cmd/conductor-cli/init.go
git commit -m "docs: add CONDUCTOR_MAIN_WORKTREE to setup script template"
```
