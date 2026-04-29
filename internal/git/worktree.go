package git

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Worktree represents a single git worktree entry.
type Worktree struct {
	Path     string // absolute path to the worktree
	Branch   string // short branch name, e.g. "main"; empty if detached
	HEAD     string // short SHA
	Detached bool
	Bare     bool
	Locked   bool
	Prunable bool
}

// FolderName returns the basename of the worktree path.
func (w Worktree) FolderName() string {
	return filepath.Base(w.Path)
}

// BranchDiffersFromFolder returns true when the branch name does not equal
// the folder name (and is not empty). Used to decide whether to show the
// branch annotation next to the path.
func (w Worktree) BranchDiffersFromFolder() bool {
	if w.Branch == "" {
		return false
	}
	return w.Branch != w.FolderName()
}

// DisplayBranch returns a human-readable branch label:
//   - branch name if attached
//   - "detached: <sha>" if detached
//   - "" if no useful annotation
func (w Worktree) DisplayBranch() string {
	if w.Detached {
		if w.HEAD != "" {
			return "detached: " + shortSHA(w.HEAD)
		}
		return "detached"
	}
	return w.Branch
}

// sanitizeBranchName turns a path basename into something safe for a git
// branch name. Replaces spaces and characters git disallows with '-'.
func sanitizeBranchName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "wt"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == ' ' || r == '~' || r == '^' || r == ':' || r == '?' || r == '*' || r == '[' || r == '\\':
			b.WriteByte('-')
		default:
			b.WriteRune(r)
		}
	}
	out := b.String()
	// Git forbids leading '-' and '.'.
	out = strings.TrimLeft(out, "-.")
	if out == "" {
		return "wt"
	}
	return out
}

func shortSHA(sha string) string {
	if len(sha) >= 7 {
		return sha[:7]
	}
	return sha
}

// List returns all worktrees for the current repo.
func List() ([]Worktree, error) {
	out, err := run("git", "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parsePorcelain(out)
}

// CurrentWorktreePath returns the absolute path of the worktree containing cwd.
func CurrentWorktreePath() (string, error) {
	out, err := run("git", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// MainWorktreePath returns the absolute path of the main worktree — i.e. the
// original clone that all linked worktrees share a `.git` directory with.
// Returns an error if the repo is bare (no main worktree exists).
func MainWorktreePath() (string, error) {
	// `git rev-parse --git-common-dir` returns the .git directory shared by
	// all worktrees. For a normal repo that's `<main>/.git`; for a bare repo
	// it's the repo root itself.
	out, err := run("git", "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", err
	}
	commonDir := strings.TrimSpace(out)
	if commonDir == "" {
		return "", errors.New("could not locate main worktree")
	}
	// Detect bare repo: `core.bare` is true.
	bareOut, _ := run("git", "config", "--get", "core.bare")
	if strings.TrimSpace(bareOut) == "true" {
		return "", errors.New("repo is bare — no main worktree to cd into")
	}
	// commonDir is `<main>/.git`; parent is the main worktree.
	return filepath.Dir(commonDir), nil
}

// DefaultBranch returns "main" or "master" depending on which exists locally.
// Falls back to "main" if neither can be determined.
func DefaultBranch() string {
	for _, candidate := range []string{"main", "master"} {
		if _, err := run("git", "rev-parse", "--verify", "--quiet", "refs/heads/"+candidate); err == nil {
			return candidate
		}
	}
	return "main"
}

// CurrentBranch returns the short branch name of HEAD (empty if detached).
func CurrentBranch() string {
	out, err := run("git", "symbolic-ref", "--short", "-q", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// InsideRepo returns nil if cwd is inside a git repo, else an error.
func InsideRepo() error {
	if _, err := run("git", "rev-parse", "--is-inside-work-tree"); err != nil {
		return errors.New("not inside a git repository")
	}
	return nil
}

// AddOptions configures a `git worktree add` call from the TUI form.
type AddOptions struct {
	Path   string // required; can be relative
	Branch string // if set, passed as -b <branch> (creates new branch)
	Base   string // if set, used as the committish arg (base the new branch on this)
}

// Add runs `git worktree add` with the given options. Returns the absolute
// path of the newly created worktree on success, along with any stderr output.
//
// Semantics: Base is always a *starting point* — the new worktree gets a
// fresh branch that starts from it. If Branch is empty, a branch name is
// derived from the path basename (matching git's own default for
// `git worktree add <path>` with no committish).
func Add(opts AddOptions) (string, string, error) {
	if opts.Path == "" {
		return "", "", errors.New("path is required")
	}
	branch := opts.Branch
	if branch == "" {
		branch = sanitizeBranchName(filepath.Base(opts.Path))
	}
	args := []string{"worktree", "add", "-b", branch, opts.Path}
	if opts.Base != "" {
		args = append(args, opts.Base)
	}
	stderr, err := runCombined("git", args...)
	if err != nil {
		return "", stderr, err
	}
	abs, absErr := filepath.Abs(opts.Path)
	if absErr != nil {
		return opts.Path, stderr, nil
	}
	return abs, stderr, nil
}

// AddPassthrough runs `git worktree add` with user-supplied args verbatim.
// Returns (resolved path of first non-flag arg, stderr, error).
func AddPassthrough(args []string) (string, string, error) {
	full := append([]string{"worktree", "add"}, args...)
	stderr, err := runCombined("git", full...)
	if err != nil {
		return "", stderr, err
	}
	// Find first non-flag arg as the path (best-effort).
	path := ""
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			path = a
			break
		}
	}
	if path == "" {
		return "", stderr, nil
	}
	abs, absErr := filepath.Abs(path)
	if absErr != nil {
		return path, stderr, nil
	}
	return abs, stderr, nil
}

// Remove runs `git worktree remove <path>` without --force. Returns stderr.
func Remove(path string) (string, error) {
	stderr, err := runCombined("git", "worktree", "remove", path)
	return stderr, err
}

// --- internal helpers ---------------------------------------------------

func run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return stdout.String(), fmt.Errorf("%s: %s", name, msg)
		}
		return stdout.String(), err
	}
	return stdout.String(), nil
}

// runCombined runs a command and returns its combined stderr+stdout so callers
// can surface git's error messages verbatim.
func runCombined(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	return strings.TrimRight(string(out), "\n"), err
}

// parsePorcelain parses `git worktree list --porcelain` output.
//
// Format (each worktree is a block separated by a blank line):
//
//	worktree /abs/path
//	HEAD abc123...
//	branch refs/heads/main
//	(or "detached")
//	(optional "bare", "locked", "prunable")
func parsePorcelain(s string) ([]Worktree, error) {
	var out []Worktree
	scanner := bufio.NewScanner(strings.NewReader(s))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	var cur *Worktree
	flush := func() {
		if cur != nil && cur.Path != "" {
			out = append(out, *cur)
		}
		cur = nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			flush()
			continue
		}
		if cur == nil {
			cur = &Worktree{}
		}
		key, val, _ := strings.Cut(line, " ")
		switch key {
		case "worktree":
			cur.Path = val
		case "HEAD":
			cur.HEAD = val
		case "branch":
			cur.Branch = strings.TrimPrefix(val, "refs/heads/")
		case "detached":
			cur.Detached = true
		case "bare":
			cur.Bare = true
		case "locked":
			cur.Locked = true
		case "prunable":
			cur.Prunable = true
		}
	}
	flush()
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
