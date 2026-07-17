package view

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kode4food/toe/internal/core"
)

// Save writes the document to its current path. Unless force is set, it
// refuses an unsafe overwrite (changed on disk, or read-only)
func (d *Document) Save(opts *Options, force bool) error {
	if !d.Loaded() {
		return nil
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
		d.buf.text.String(), d.lineEnding, opts, d.editorConfig,
	)
	if err := d.applySaveText(text); err != nil {
		return err
	}
	var data []byte
	if d.hasBOM {
		data = append([]byte{0xef, 0xbb, 0xbf}, []byte(text)...)
	} else {
		data = []byte(text)
	}
	var backup string
	if opts.AtomicSave {
		if _, statErr := os.Stat(path); statErr == nil {
			if b, err := renameToBackup(path, dir); err == nil {
				backup = b
			}
		}
	}
	var err error
	if opts.AtomicSave {
		err = atomicWrite(path, dir, data)
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
	d.buf.savePoint = d.buf.history.CurrentRevision()
	d.buf.unsaved = false
	return nil
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
	d.hasBOM = hasBOMBytes(data)
	if d.hasBOM {
		data = data[3:]
	}
	oldText := d.buf.text
	newText := string(data)
	text := core.NewRope(newText)
	cs, err := diffChangeSet(oldText, newText)
	if err != nil {
		return err
	}
	selections := mapSelections(d.buf.selections, cs, text.LenChars())
	d.buf.Lock()
	d.buf.text = text
	d.buf.version++
	d.buf.selections = selections
	d.buf.Unlock()
	d.markAllDirty()
	d.buf.savePoint = d.buf.history.CurrentRevision()
	d.buf.unsaved = false
	d.refreshDiskSnapshot()
	return nil
}

func (d *Document) applySaveText(text string) error {
	oldText := d.buf.text
	cs, err := diffChangeSet(oldText, text)
	if err != nil || cs.Empty() {
		return err
	}
	sel := d.Selection()
	tx := core.NewTransaction(oldText).WithChanges(cs).WithSelection(sel)
	st := core.State{Doc: oldText, Selection: sel}
	if err := d.buf.history.CommitRevision(tx, st); err != nil {
		return err
	}
	newText, err := tx.Apply(oldText)
	if err != nil {
		return err
	}
	for vid, sel := range d.buf.selections {
		if mapped, err := sel.Map(cs); err == nil {
			d.buf.selections[vid] = mapped
		}
	}
	if mapped, err := d.buf.lastSel.Map(cs); err == nil {
		d.buf.lastSel = mapped
	}
	d.buf.Lock()
	d.buf.text = newText
	d.buf.version++
	d.buf.unsaved = true
	d.buf.modified = true
	d.buf.Unlock()
	d.markAllDirty()
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

// DocumentDisplayName returns a short display name for a file path,
// or ScratchBufferName if path is empty
func DocumentDisplayName(path string) string {
	if path == "" {
		return ScratchBufferName
	}
	return filepath.Base(path)
}

// DocumentRelativeName returns path relative to basedir,
// falling back to the absolute path on error
func DocumentRelativeName(path, basedir string) string {
	if path == "" {
		return ScratchBufferName
	}
	rel, err := filepath.Rel(basedir, path)
	if err != nil {
		return path
	}
	if !strings.HasPrefix(rel, "..") {
		return rel
	}
	return path
}

func renameToBackup(path, dir string) (string, error) {
	f, err := os.CreateTemp(dir, filepath.Base(path)+".bck-*")
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

func atomicWrite(path, dir string, data []byte) error {
	f, err := os.CreateTemp(dir, ".toe-save-*")
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
	return os.Rename(tmp, path)
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
