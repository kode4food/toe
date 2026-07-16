package vcs

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"

	"github.com/kode4food/toe/internal/view"
)

// Git reads HEAD state through go-git in-process and reports working-tree
// status by shelling out to the git binary found on PATH
type Git struct{}

var (
	ErrGitCommand   = errors.New("git command failed")
	ErrGitBadStatus = errors.New("unparsable git status output")
)

var _ Provider = Git{}

// DiffBase returns the HEAD contents of path. No eol/ident smudge filtering
// is applied; .gitattributes eol conversion may cause phantom diffs
func (Git) DiffBase(path string) ([]byte, error) {
	path = realPath(path)
	repo, err := openRepo(path)
	if err != nil {
		return nil, err
	}
	root, err := repoRoot(repo)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}
	ref, err := repo.Head()
	if err != nil {
		return nil, err
	}
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, err
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}
	file, err := tree.File(filepath.ToSlash(rel))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGitCommand, err)
	}
	content, err := file.Contents()
	if err != nil {
		return nil, err
	}
	return []byte(content), nil
}

// HeadName returns the current branch name, or a short commit hash when the
// head is detached
func (Git) HeadName(path string) (string, error) {
	repo, err := openRepo(path)
	if err != nil {
		return "", err
	}
	ref, err := repo.Head()
	if err != nil {
		return "", err
	}
	if ref.Name().IsBranch() {
		return ref.Name().Short(), nil
	}
	return ref.Hash().String()[:8], nil
}

// HeadID returns the full current HEAD revision
func (Git) HeadID(path string) (string, error) {
	repo, err := openRepo(path)
	if err != nil {
		return "", err
	}
	ref, err := repo.Head()
	if err != nil {
		return "", err
	}
	return ref.Hash().String(), nil
}

// ChangedFiles reports the working-tree changes for the repository containing
// cwd, with absolute paths. It prefers the git binary (fast) and falls back to
// go-git's in-process status walk when git is not on PATH
func (Git) ChangedFiles(cwd string) ([]view.FileChange, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return changedFilesGoGit(cwd)
	}
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

// go-git reports the same X/Y status codes as porcelain but does not detect
// renames, so a moved file shows as add + delete
func changedFilesGoGit(cwd string) ([]view.FileChange, error) {
	repo, err := git.PlainOpenWithOptions(
		cwd, &git.PlainOpenOptions{DetectDotGit: true},
	)
	if err != nil {
		return nil, err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return nil, err
	}
	root := realPath(wt.Filesystem.Root())
	st, err := wt.Status()
	if err != nil {
		return nil, err
	}
	var changes []view.FileChange
	for p, fs := range st {
		x, y := byte(fs.Staging), byte(fs.Worktree)
		if x == ' ' && y == ' ' {
			continue
		}
		kind := changeKind(x, y)
		fc := view.FileChange{
			Kind: kind, Path: filepath.Join(root, filepath.FromSlash(p)),
		}
		if kind == view.FileChangeRenamed && fs.Extra != "" {
			fc.FromPath = filepath.Join(root, filepath.FromSlash(fs.Extra))
		}
		changes = append(changes, fc)
	}
	return changes, nil
}

// parseGitStatus decodes NUL-terminated porcelain entries; rename entries
// carry the original path in an extra field
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
		kind := changeKind(x, y)
		fc := view.FileChange{
			Kind: kind,
			Path: filepath.Join(root, filepath.FromSlash(entry[3:])),
		}
		if kind == view.FileChangeRenamed {
			if i+1 >= len(fields) {
				return nil, fmt.Errorf("%w: %q", ErrGitBadStatus, entry)
			}
			i++
			fc.FromPath = filepath.Join(root, filepath.FromSlash(fields[i]))
		}
		changes = append(changes, fc)
	}
	return changes, nil
}

func changeKind(x, y byte) view.FileChangeKind {
	switch {
	case x == '?' && y == '?':
		return view.FileChangeUntracked
	case gitConflict(x, y):
		return view.FileChangeConflict
	case x == 'R' || y == 'R':
		return view.FileChangeRenamed
	case x == 'D' || y == 'D':
		return view.FileChangeDeleted
	case x == 'A':
		return view.FileChangeAdded
	default:
		return view.FileChangeModified
	}
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

func openRepo(path string) (*git.Repository, error) {
	return git.PlainOpenWithOptions(
		filepath.Dir(path), &git.PlainOpenOptions{DetectDotGit: true},
	)
}

func repoRoot(repo *git.Repository) (string, error) {
	wt, err := repo.Worktree()
	if err != nil {
		return "", err
	}
	return realPath(wt.Filesystem.Root()), nil
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
