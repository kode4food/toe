package vcs

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/kode4food/toe/internal/view"
)

type (
	// Git reads HEAD and index contents through one long-running cat-file
	// process per repository, and shells out to git for everything else
	Git struct {
		mu       sync.Mutex
		dirs     map[string]*gitRepo
		repos    map[string]*gitRepo
		realDirs map[string]string
	}

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

const (
	gitIgnoreName = ".gitignore"

	// what rev-parse --abbrev-ref reports when no branch is checked out
	detachedHead = "HEAD"

	gitIndexName = "index"
	repoDirLines = 2
)

var (
	ErrGitCommand   = errors.New("git command failed")
	ErrGitBadStatus = errors.New("unparsable git status output")
	ErrGitBadBatch  = errors.New("unparsable git cat-file output")
)

var _ Provider = (*Git)(nil)

// DiffBase returns the HEAD contents of path. No eol/ident smudge filtering
// is applied. A .gitattributes eol conversion may cause phantom diffs
func (g *Git) DiffBase(path string) ([]byte, error) {
	path = g.realPath(path)
	repo, err := g.repo(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(repo.root, path)
	if err != nil {
		return nil, err
	}
	return g.readBlob(repo, "HEAD:"+filepath.ToSlash(rel))
}

// HeadName returns the current branch name, or a short commit hash when the
// head is detached
func (g *Git) HeadName(path string) (string, error) {
	dir := filepath.Dir(path)
	out, err := runGit(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	if name := strings.TrimSpace(string(out)); name != detachedHead {
		return name, nil
	}
	short, err := runGit(dir, "rev-parse", "--short=8", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(short)), nil
}

// HeadID returns the full current HEAD revision
func (g *Git) HeadID(path string) (string, error) {
	out, err := runGit(filepath.Dir(path), "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ChangedFiles reports the working-tree changes for the repository containing
// cwd, with absolute paths
func (g *Git) ChangedFiles(cwd string) ([]view.FileChange, error) {
	repo, err := g.repo(cwd)
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
		root:   repo.root,
		output: string(out),
	})
}

// Stage adds the working-tree state of path to the staging area, recording an
// added, modified, or deleted file alike
func (g *Git) Stage(cwd, path string) error {
	_, err := runGit(cwd, "add", "--", path)
	return err
}

// Unstage drops path from the staging area, leaving the working tree alone
func (g *Git) Unstage(cwd, path string) error {
	if _, err := runGit(cwd, "restore", "--staged", "--", path); err == nil {
		return nil
	}
	// a repository without a first commit has no HEAD to restore from, so the
	// index entry itself is dropped
	_, err := runGit(cwd, "rm", "--cached", "-q", "--", path)
	return err
}

// Discard restores the working-tree state of path from the index. An untracked
// file has no index state to restore, so it is deleted instead
func (g *Git) Discard(cwd, path string) error {
	if !tracked(cwd, path) {
		return os.Remove(path)
	}
	_, err := runGit(cwd, "restore", "--worktree", "--", path)
	return err
}

// Ignore appends path to the repository's root .gitignore, anchored so it names
// that one file rather than every file sharing its name
func (g *Git) Ignore(cwd, path string) error {
	repo, err := g.repo(cwd)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(repo.root, g.realPath(path))
	if err != nil {
		return err
	}
	return appendIgnore(
		filepath.Join(repo.root, gitIgnoreName), "/"+filepath.ToSlash(rel),
	)
}

// Tracked reports whether the repository containing cwd has path in its index
func (g *Git) Tracked(cwd, path string) bool {
	return tracked(cwd, path)
}

// IndexText returns the staged blob for path, erroring when the file has no
// entry in the index
func (g *Git) IndexText(cwd, path string) ([]byte, error) {
	path = g.realPath(path)
	repo, err := g.repo(cwd)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(repo.root, path)
	if err != nil {
		return nil, err
	}
	return g.readBlob(repo, ":"+filepath.ToSlash(rel))
}

// Close stops the cat-file processes
func (g *Git) Close() {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, r := range g.repos {
		r.stopBatch()
	}
}

// a directory's repository is resolved once; a failed lookup is not cached, so
// a later git init is picked up
func (g *Git) repo(dir string) (*gitRepo, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if r, ok := g.dirs[dir]; ok {
		return r, nil
	}
	out, err := runGit(
		dir, "rev-parse", "--show-toplevel", "--absolute-git-dir",
	)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != repoDirLines {
		return nil, fmt.Errorf("%w: %s", ErrGitCommand, out)
	}
	if g.dirs == nil {
		g.dirs = map[string]*gitRepo{}
		g.repos = map[string]*gitRepo{}
	}
	r, ok := g.repos[lines[0]]
	if !ok {
		r = &gitRepo{
			root:  lines[0],
			index: filepath.Join(lines[1], gitIndexName),
			blobs: map[string][]byte{},
		}
		g.repos[r.root] = r
	}
	g.dirs[dir] = r
	return r, nil
}

// a directory's real path is resolved once, like its repository, so only the
// file itself is checked for a symlink on each call
func (g *Git) realPath(path string) string {
	info, err := os.Lstat(path)
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			return resolved
		}
		return path
	}
	dir := filepath.Dir(path)
	g.mu.Lock()
	defer g.mu.Unlock()
	if resolved, ok := g.realDirs[dir]; ok {
		return filepath.Join(resolved, filepath.Base(path))
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return path
	}
	if g.realDirs == nil {
		g.realDirs = map[string]string{}
	}
	g.realDirs[dir] = resolved
	return filepath.Join(resolved, filepath.Base(path))
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

// an ignore file that does not end in a newline would swallow the pattern into
// its last line
func appendIgnore(file, pattern string) error {
	data, err := os.ReadFile(file)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if slices.Contains(strings.Split(string(data), "\n"), pattern) {
		return nil
	}
	out := string(data)
	if out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return os.WriteFile(file, []byte(out+pattern+"\n"), 0o644)
}

func tracked(cwd, path string) bool {
	_, err := runGit(cwd, "ls-files", "--error-unmatch", "--", path)
	return err == nil
}

func runGit(dir string, args ...string) ([]byte, error) {
	// avoids status's index stat-refresh write re-triggering our watcher
	all := append([]string{"-C", dir, "--no-optional-locks"}, args...)
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
