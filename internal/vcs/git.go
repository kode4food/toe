package vcs

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kode4food/toe/internal/view"
)

// Git provides diff bases and file status by shelling out to the git binary
// found on PATH
type Git struct{}

var (
	ErrGitCommand   = errors.New("git command failed")
	ErrGitBadStatus = errors.New("unparsable git status output")
)

var _ Provider = Git{}

// DiffBase returns the HEAD contents of path. The --filters flag applies the
// work-tree conversions (eol, ident) git would perform on checkout, so the
// result matches what an unedited file contains on disk
func (Git) DiffBase(path string) ([]byte, error) {
	path = realPath(path)
	dir := filepath.Dir(path)
	root, err := gitRoot(dir)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}
	return runGit(dir, "cat-file", "--filters", "HEAD:"+filepath.ToSlash(rel))
}

// HeadName returns the current branch name, or a short commit hash when the
// head is detached
func (Git) HeadName(path string) (string, error) {
	dir := filepath.Dir(path)
	out, err := runGit(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(string(out))
	if name != "HEAD" {
		return name, nil
	}
	out, err = runGit(dir, "rev-parse", "--short=8", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// HeadID returns the full current HEAD revision
func (Git) HeadID(path string) (string, error) {
	dir := filepath.Dir(path)
	out, err := runGit(dir, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ChangedFiles emulates `git status` for the repository containing cwd,
// reporting paths as absolute
func (Git) ChangedFiles(cwd string) ([]view.FileChange, error) {
	root, err := gitRoot(cwd)
	if err != nil {
		return nil, err
	}
	out, err := runGit(
		cwd, "status", "--porcelain", "-z", "--untracked-files=all",
		"--find-renames",
	)
	if err != nil {
		return nil, err
	}
	return parseGitStatus(root, string(out))
}

// parseGitStatus decodes `git status --porcelain -z` output. Each entry is
// "XY path" (X: index status, Y: work-tree status) terminated by NUL; rename
// entries carry the original path as an extra NUL-terminated field
func parseGitStatus(root, out string) ([]view.FileChange, error) {
	var changes []view.FileChange
	fields := strings.Split(out, "\x00")
	for i := 0; i < len(fields); i++ {
		entry := fields[i]
		if entry == "" {
			continue
		}
		if len(entry) < 4 || entry[2] != ' ' {
			return nil, fmt.Errorf("%w: %q", ErrGitBadStatus, entry)
		}
		x, y := entry[0], entry[1]
		path := filepath.Join(root, filepath.FromSlash(entry[3:]))
		switch {
		case x == '?' && y == '?':
			changes = append(changes, view.FileChange{
				Kind: view.FileChangeUntracked, Path: path,
			})
		case gitConflict(x, y):
			changes = append(changes, view.FileChange{
				Kind: view.FileChangeConflict, Path: path,
			})
		case x == 'R' || y == 'R':
			if i+1 >= len(fields) {
				return nil, fmt.Errorf("%w: %q", ErrGitBadStatus, entry)
			}
			i++
			from := filepath.Join(root, filepath.FromSlash(fields[i]))
			changes = append(changes, view.FileChange{
				Kind: view.FileChangeRenamed, Path: path, FromPath: from,
			})
		case x == 'D' || y == 'D':
			changes = append(changes, view.FileChange{
				Kind: view.FileChangeDeleted, Path: path,
			})
		case x == 'A':
			changes = append(changes, view.FileChange{
				Kind: view.FileChangeAdded, Path: path,
			})
		default:
			changes = append(changes, view.FileChange{
				Kind: view.FileChangeModified, Path: path,
			})
		}
	}
	return changes, nil
}

func gitConflict(x, y byte) bool {
	return x == 'U' || y == 'U' || (x == 'D' && y == 'D') ||
		(x == 'A' && y == 'A')
}

// realPath resolves symlinks so paths compare cleanly against the repo root git
// reports (macOS /var vs /private/var, for example)
func realPath(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return path
}

func gitRoot(dir string) (string, error) {
	out, err := runGit(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func runGit(dir string, args ...string) ([]byte, error) {
	all := append([]string{"-C", dir}, args...)
	out, err := exec.Command("git", all...).Output()
	if err != nil {
		exitErr, ok := errors.AsType[*exec.ExitError](err)
		if ok && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf(
				"%w: %s", ErrGitCommand,
				strings.TrimSpace(string(exitErr.Stderr)),
			)
		}
		return nil, fmt.Errorf("%w: %v", ErrGitCommand, err)
	}
	return out, nil
}
