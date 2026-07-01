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
	if text != d.buf.text.String() {
		d.buf.Lock()
		d.buf.text = core.NewRope(text)
		d.buf.version++
		d.buf.Unlock()
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
	d.buf.modified = false
	return nil
}

// Reload replaces the document text with the current file contents on disk
// All per-view selections are reset to the start of the document
func (d *Document) Reload() error {
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
	d.buf.Lock()
	d.buf.text = core.NewRope(string(data))
	d.buf.version++
	d.buf.Unlock()
	d.buf.modified = false
	d.buf.history = core.NewHistory()
	for vid := range d.buf.selections {
		sel, _ := core.NewSelection([]core.Range{core.PointRange(0)}, 0)
		d.buf.selections[vid] = sel
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

// hasBOMBytes reports whether data begins with the UTF-8 BOM (0xef 0xbb 0xbf)
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
