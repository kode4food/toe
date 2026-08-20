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

type (
	// Git reads HEAD state through go-git in-process and reports working-tree
	// status by shelling out to the git binary found on PATH
	Git struct{}

	// statusCode is a porcelain entry's staged (X) and unstaged (Y) codes
	statusCode struct {
		staged   byte
		unstaged byte
	}

	gitStatus struct {
		staged      view.FileChangeKind
		unstaged    view.FileChangeKind
		hasStaged   bool
		hasUnstaged bool
	}
)

var (
	ErrGitCommand   = errors.New("git command failed")
	ErrGitBadStatus = errors.New("unparsable git status output")
)

var _ Provider = Git{}

// DiffBase returns the HEAD contents of path. No eol/ident smudge filtering
// is applied. A .gitattributes eol conversion may cause phantom diffs
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
	return parseGitStatus(parseGitStatusArgs{
		root:   root,
		output: string(out),
	})
}

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
		status := splitChangeKind(statusCode{staged: x, unstaged: y})
		fc := view.FileChange{
			Path: filepath.Join(root, filepath.FromSlash(p)),
		}
		if renamed(status) && fs.Extra != "" {
			fc.FromPath = filepath.Join(root, filepath.FromSlash(fs.Extra))
		}
		changes = appendChanges(changes, status, fc)
	}
	return changes, nil
}

type parseGitStatusArgs struct {
	root   string
	output string
}

func parseGitStatus(args parseGitStatusArgs) ([]view.FileChange, error) {
	root := args.root
	var changes []view.FileChange
	fields := strings.Split(args.output, "\x00")
	for i := 0; i < len(fields); i++ {
		entry := fields[i]
		if entry == "" {
			continue
		}
		if len(entry) < 4 || entry[2] != ' ' {
			return nil, fmt.Errorf("%w: %q", ErrGitBadStatus, entry)
		}
		x, y := entry[0], entry[1]
		st := splitChangeKind(statusCode{staged: x, unstaged: y})
		fc := view.FileChange{
			Path: filepath.Join(root, filepath.FromSlash(entry[3:])),
		}
		if renamed(st) {
			if i+1 >= len(fields) {
				return nil, fmt.Errorf("%w: %q", ErrGitBadStatus, entry)
			}
			i++
			fc.FromPath = filepath.Join(root, filepath.FromSlash(fields[i]))
		}
		changes = appendChanges(changes, st, fc)
	}
	return changes, nil
}

func splitChangeKind(code statusCode) gitStatus {
	x := code.staged
	y := code.unstaged
	switch {
	case x == '?' && y == '?':
		return gitStatus{
			unstaged: view.FileChangeUntracked, hasUnstaged: true,
		}
	case isGitConflict(statusCode{staged: x, unstaged: y}):
		return gitStatus{
			unstaged: view.FileChangeConflict, hasUnstaged: true,
		}
	}
	var st gitStatus
	if x != ' ' && x != '?' {
		st.staged, st.hasStaged = changeKind(x), true
	}
	if y != ' ' && y != '?' {
		st.unstaged, st.hasUnstaged = changeKind(y), true
	}
	return st
}

func changeKind(c byte) view.FileChangeKind {
	switch c {
	case 'R':
		return view.FileChangeRenamed
	case 'D':
		return view.FileChangeDeleted
	case 'A', 'C':
		return view.FileChangeAdded
	default:
		return view.FileChangeModified
	}
}

func renamed(st gitStatus) bool {
	return (st.hasStaged && st.staged == view.FileChangeRenamed) ||
		(st.hasUnstaged && st.unstaged == view.FileChangeRenamed)
}

func appendChanges(
	changes []view.FileChange, st gitStatus, base view.FileChange,
) []view.FileChange {
	if st.hasStaged {
		fc := base
		fc.Kind, fc.Staged = st.staged, true
		changes = append(changes, fc)
	}
	if st.hasUnstaged {
		fc := base
		fc.Kind, fc.Staged = st.unstaged, false
		changes = append(changes, fc)
	}
	return changes
}

func isGitConflict(code statusCode) bool {
	x := code.staged
	y := code.unstaged
	return x == 'U' || y == 'U' || (x == 'D' && y == 'D') ||
		(x == 'A' && y == 'A')
}

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
