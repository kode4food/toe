package view

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kode4food/toe/internal/core"
)

// saveTarget is a file being written and the directory its temporaries go in
type saveTarget struct {
	path string
	dir  string
}

// Save writes the document to its current path. Unless force is set, it refuses
// an unsafe overwrite (changed on disk, or read-only). A log has no path of its
// own, so it can only be copied out with WriteCopy
func (d *Document) Save(opts *Options, force bool) error {
	if !d.Loaded() {
		return nil
	}
	if d.identity.docType == DocTypeLog {
		return ErrReadOnly
	}
	path := d.Path()
	if path == "" {
		return ErrDocumentNoPath
	}
	if !force {
		if err := d.checkSafeToOverwrite(path); err != nil {
			return err
		}
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	text := prepareSaveText(
		d.content.text.String(), d.format.lineEnding, opts,
		d.format.editorConfig,
	)
	if err := d.applySaveText(text); err != nil {
		return err
	}
	var data []byte
	if d.format.hasBOM {
		data = append([]byte{0xef, 0xbb, 0xbf}, []byte(text)...)
	} else {
		data = []byte(text)
	}
	target := saveTarget{path: path, dir: dir}
	var backup string
	if opts.AtomicSave {
		if _, statErr := os.Stat(path); statErr == nil {
			if b, err := renameToBackup(target); err == nil {
				backup = b
			}
		}
	}
	var err error
	if opts.AtomicSave {
		err = atomicWrite(target, data)
	} else {
		err = writeFileSynced(path, data, 0o644)
	}
	if backup != "" {
		if err != nil {
			_ = os.Rename(backup, path)
		} else {
			_ = os.Remove(backup)
		}
	}
	if err != nil {
		return err
	}
	d.refreshDiskSnapshot()
	d.markSaved()
	return nil
}

// WriteCopy writes the document's text to path, leaving the document itself
// alone: it keeps its own path, name, and modification state
func (d *Document) WriteCopy(path string, opts *Options) error {
	d.ensureLoaded()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	text := prepareSaveText(
		d.Text().String(), d.format.lineEnding, opts, d.format.editorConfig,
	)
	return writeFileSynced(path, []byte(text), 0o644)
}

// Reload replaces the document text with the current file contents on disk
// All per-view selections are reset to the start of the document
func (d *Document) Reload() error {
	return d.reloadPreservingSelections()
}

func (d *Document) reloadPreservingSelections() error {
	d.ensureLoaded()
	path := d.Path()
	if path == "" {
		return ErrDocumentNoPath
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	d.format.hasBOM = hasBOMBytes(data)
	if d.format.hasBOM {
		data = data[3:]
	}
	oldText := d.content.text
	newText := string(data)
	text := core.NewRope(newText)
	cs, err := diffChangeSet(oldText, newText)
	if err != nil {
		return err
	}
	selections := mapSelections(d.views.selections, cs, text.LenChars())
	d.content.Lock()
	d.content.text = text
	d.content.version++
	d.content.Unlock()
	d.views.selections = selections
	d.MarkDirty()
	d.markSaved()
	d.refreshDiskSnapshot()
	return nil
}

func (d *Document) applySaveText(text string) error {
	oldText := d.content.text
	cs, err := diffChangeSet(oldText, text)
	if err != nil || cs.Empty() {
		return err
	}
	sel := d.Selection()
	tx := core.NewTransaction(oldText).WithChanges(cs).WithSelection(sel)
	st := core.State{Doc: oldText, Selection: sel}
	if err := d.edits.history.CommitRevision(tx, st); err != nil {
		return err
	}
	newText, err := tx.Apply(oldText)
	if err != nil {
		return err
	}
	for vid, sel := range d.views.selections {
		if mapped, err := sel.Map(cs); err == nil {
			d.views.selections[vid] = mapped
		}
	}
	if mapped, err := d.views.lastSelection.Map(cs); err == nil {
		d.views.lastSelection = mapped
	}
	d.content.Lock()
	d.content.text = newText
	d.content.version++
	d.content.Unlock()
	d.edits.changedSinceAccess = true
	d.MarkDirty()
	return nil
}

func (d *Document) checkSafeToOverwrite(path string) error {
	if info, err := os.Stat(path); err == nil &&
		info.Mode().Perm()&0o200 == 0 {
		return ErrFileReadOnly
	}
	if _, changed := d.diskChanged(); changed {
		return ErrFileChangedOnDisk
	}
	return nil
}

func (d *Document) markSaved() {
	d.edits.savePoint = d.edits.history.CurrentRevision()
}

// DocumentRelativeNameArgs is a document path and the directory to make it
// relative to
type DocumentRelativeNameArgs struct {
	Path    string
	BaseDir string
}

// DocumentRelativeName returns Path relative to BaseDir, falling back to the
// absolute path on error
func DocumentRelativeName(args DocumentRelativeNameArgs) string {
	path := args.Path
	if path == "" {
		return ScratchBufferName
	}
	rel, err := filepath.Rel(args.BaseDir, path)
	if err != nil {
		return path
	}
	if !strings.HasPrefix(rel, "..") {
		return rel
	}
	return path
}

func renameToBackup(target saveTarget) (string, error) {
	path := target.path
	f, err := os.CreateTemp(target.dir, filepath.Base(path)+".bck-*")
	if err != nil {
		return "", err
	}
	tmp := f.Name()
	if err := f.Close(); err != nil {
		return "", err
	}
	if err := os.Remove(tmp); err != nil {
		return "", err
	}
	if err := os.Rename(path, tmp); err != nil {
		return "", err
	}
	return tmp, nil
}

func atomicWrite(target saveTarget, data []byte) error {
	f, err := os.CreateTemp(target.dir, ".toe-save-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	if _, err = f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err = f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, target.path)
}

func writeFileSynced(path string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err = f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
