package view

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2/lexers"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view/config"
	"github.com/kode4food/toe/internal/view/language"
)

// Save writes the document to its current path
func (d *Document) Save(opts *Options) error {
	if !d.Loaded() {
		return nil
	}
	path := d.Path()
	if path == "" {
		return ErrDocumentNoPath
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
	var err error
	if opts.AtomicSave {
		err = atomicWrite(path, dir, data)
	} else {
		err = os.WriteFile(path, data, 0o644)
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
	if err = f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func diffChangeSet(oldText core.Rope, newText string) (core.ChangeSet, error) {
	oldRunes := []rune(oldText.String())
	newRunes := []rune(newText)
	pfx := commonPrefix(oldRunes, newRunes)
	sfx := commonSuffix(oldRunes[pfx:], newRunes[pfx:])
	from := pfx
	to := len(oldRunes) - sfx
	repl := string(newRunes[pfx : len(newRunes)-sfx])
	return core.NewChangeSetFromChanges(oldText, []core.Change{
		core.TextChange(from, to, repl),
	})
}

func mapSelections(
	selections map[Id]core.Selection, cs core.ChangeSet, n int,
) map[Id]core.Selection {
	out := make(map[Id]core.Selection, len(selections))
	for vid, sel := range selections {
		out[vid] = mapSelection(sel, cs, n)
	}
	return out
}

func mapSelection(sel core.Selection, cs core.ChangeSet, n int) core.Selection {
	out, err := sel.Map(cs)
	if err == nil {
		return out
	}
	ranges := sel.Ranges()
	for i, r := range ranges {
		ranges[i] = core.NewRange(clipPos(r.Anchor, n), clipPos(r.Head, n))
	}
	out, err = core.NewSelection(ranges, sel.PrimaryIndex())
	if err != nil {
		return core.PointSelection(clipPos(sel.Primary().Head, n))
	}
	return out
}

func commonPrefix(a, b []rune) int {
	n := min(len(a), len(b))
	for i := range n {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

func commonSuffix(a, b []rune) int {
	n := min(len(a), len(b))
	for i := range n {
		if a[len(a)-1-i] != b[len(b)-1-i] {
			return i
		}
	}
	return n
}

func clipPos(pos, n int) int {
	return min(max(pos, 0), n)
}

// detectLang returns a Chroma-compatible language name for the given file path
// and content. Falls back to "text" if no match is found
func detectLang(path, content string) string {
	if lang, ok := language.DetectLanguage(path, content); ok {
		return lang
	}
	if lex := lexers.Match(path); lex != nil {
		return strings.ToLower(lex.Config().Name)
	}
	if lex := lexers.Analyse(content); lex != nil {
		return strings.ToLower(lex.Config().Name)
	}
	return "text"
}

func defaultLineEnding(le core.LineEnding) core.LineEnding {
	if le == "" {
		return core.NativeLineEnding()
	}
	return le
}

func prepareSaveText(
	s string, le core.LineEnding, opts *Options, ec *config.EditorConfig,
) string {
	trim := opts.TrimTrailingWS
	if ec != nil && ec.TrimTrailingWhitespace != nil {
		trim = *ec.TrimTrailingWhitespace
	}
	insert := opts.InsertFinalNewline
	if ec != nil && ec.InsertFinalNewline != nil {
		insert = *ec.InsertFinalNewline
	}
	if trim {
		s = trimTrailingWhitespace(s)
	}
	if opts.TrimFinalNewlines {
		s = trimFinalNewlines(s)
	}
	if insert && s != "" {
		if _, ok := core.GetLineEndingOfString(s); !ok {
			s += string(le)
		}
	}
	return s
}

func trimTrailingWhitespace(s string) string {
	lines := strings.SplitAfter(s, "\n")
	var b strings.Builder
	for _, line := range lines {
		ending := ""
		body := line
		if strings.HasSuffix(line, "\r\n") {
			ending = "\r\n"
			body = strings.TrimSuffix(line, ending)
		} else if strings.HasSuffix(line, "\n") {
			ending = "\n"
			body = strings.TrimSuffix(line, ending)
		}
		b.WriteString(strings.TrimRight(body, " \t"))
		b.WriteString(ending)
	}
	return b.String()
}

func hasBOMBytes(data []byte) bool {
	return len(data) >= 3 &&
		data[0] == 0xef && data[1] == 0xbb && data[2] == 0xbf
}

func trimFinalNewlines(s string) string {
	total := 0
	final := 0
	for {
		le, ok := core.GetLineEndingOfString(s[:len(s)-total])
		if !ok {
			break
		}
		n := len(le)
		total += n
		final = n
	}
	if total == final {
		return s
	}
	return s[:len(s)-total+final]
}
