package vcs

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type (
	// blobs are keyed by object id, which addresses their content, so a cached
	// blob never goes stale
	gitRepo struct {
		root  string
		index string
		stat  os.FileInfo
		batch *catFile
		blobs map[string][]byte
	}

	catFile struct {
		cmd *exec.Cmd
		in  io.WriteCloser
		out *bufio.Reader
	}

	catFileHeader struct {
		oid  string
		kind string
		size int
	}
)

const (
	blobKind            = "blob"
	maxCachedBlobs      = 64
	catFileHeaderFields = 3
	catFileTrailer      = 1
)

// cat-file cannot take a newline in an object name, and a failed process has
// nothing to serve, so both fall back to a one-shot git show
func (g *Git) readBlob(repo *gitRepo, spec string) ([]byte, error) {
	if strings.ContainsRune(spec, '\n') {
		return runGit(repo.root, "show", spec)
	}
	data, err := g.batchBlob(repo, spec)
	if err != nil && !errors.Is(err, ErrGitCommand) {
		return runGit(repo.root, "show", spec)
	}
	return data, err
}

func (g *Git) batchBlob(repo *gitRepo, spec string) ([]byte, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	data, err := repo.blob(spec)
	if err != nil && !errors.Is(err, ErrGitCommand) {
		repo.stopBatch()
	}
	return data, err
}

func (g *gitRepo) blob(spec string) ([]byte, error) {
	if err := g.ensureBatch(); err != nil {
		return nil, err
	}
	h, err := g.batch.request("info " + spec)
	if err != nil {
		return nil, err
	}
	if h.kind != blobKind {
		return nil, fmt.Errorf("%w: %s", ErrGitCommand, spec)
	}
	if data, ok := g.blobs[h.oid]; ok {
		return data, nil
	}
	if h, err = g.batch.request("contents " + h.oid); err != nil {
		return nil, err
	}
	data := make([]byte, h.size+catFileTrailer)
	if _, err := io.ReadFull(g.batch.out, data); err != nil {
		return nil, err
	}
	data = data[:h.size]
	if len(g.blobs) >= maxCachedBlobs {
		clear(g.blobs)
	}
	g.blobs[h.oid] = data
	return data, nil
}

// cat-file loads the index once and never rereads it, so the process restarts
// whenever git rewrites the index file
func (g *gitRepo) ensureBatch() error {
	info, _ := os.Stat(g.index)
	if g.batch != nil && !g.indexMoved(info) {
		return nil
	}
	g.stopBatch()
	batch, err := startCatFile(g.root)
	if err != nil {
		return err
	}
	g.batch = batch
	g.stat = info
	return nil
}

func (g *gitRepo) indexMoved(cur os.FileInfo) bool {
	old := g.stat
	if old == nil || cur == nil {
		return old != cur
	}
	return !os.SameFile(old, cur) || !old.ModTime().Equal(cur.ModTime()) ||
		old.Size() != cur.Size()
}

func (g *gitRepo) stopBatch() {
	if g.batch == nil {
		return
	}
	_ = g.batch.in.Close()
	_ = g.batch.cmd.Process.Kill()
	_ = g.batch.cmd.Wait()
	g.batch = nil
}

func (c *catFile) request(line string) (catFileHeader, error) {
	if _, err := io.WriteString(c.in, line+"\n"); err != nil {
		return catFileHeader{}, err
	}
	head, err := c.out.ReadString('\n')
	if err != nil {
		return catFileHeader{}, err
	}
	head = strings.TrimSuffix(head, "\n")
	if strings.HasSuffix(head, " missing") ||
		strings.HasSuffix(head, " ambiguous") {
		return catFileHeader{}, fmt.Errorf("%w: %s", ErrGitCommand, head)
	}
	fields := strings.Fields(head)
	if len(fields) != catFileHeaderFields {
		return catFileHeader{}, fmt.Errorf("%w: %q", ErrGitBadBatch, head)
	}
	size, err := strconv.Atoi(fields[2])
	if err != nil {
		return catFileHeader{}, fmt.Errorf("%w: %q", ErrGitBadBatch, head)
	}
	return catFileHeader{oid: fields[0], kind: fields[1], size: size}, nil
}

func startCatFile(root string) (*catFile, error) {
	cmd := exec.Command("git", "-C", root, "cat-file", "--batch-command")
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &catFile{cmd: cmd, in: in, out: bufio.NewReader(out)}, nil
}
