package ui

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

type (
	globalSearchSource struct {
		pickerMeta
		query     string
		smartCase bool
		openDocs  []docSnap
	}

	globalSearchPayload struct {
		path string
		line int
	}

	globalSearcher struct {
		ch       chan PickerItem
		done     chan struct{}
		re       *regexp.Regexp
		root     string
		openDocs []docSnap
	}
)

func (g *globalSearchSource) Search(query string) {
	g.query = query
}

func (g *globalSearchSource) Load(
	e *view.Editor,
) ([]PickerItem, <-chan PickerItem, StopFunc) {
	if g.query == "" {
		return nil, nil, func() {}
	}
	g.smartCase = e.Options().SearchSmartCase
	g.openDocs = nil
	for _, doc := range e.AllDocuments() {
		if path := doc.Path(); path != "" {
			g.openDocs = append(g.openDocs, docSnap{path, doc.Text().String()})
		}
	}
	ch, cancel := globalSearchQuery(e.Cwd(), g.openDocs, g.query, g.smartCase)
	return nil, ch, cancel
}

func (g *globalSearchSource) Accept(e *view.Editor, item PickerItem) {
	payload, ok := item.Payload.(globalSearchPayload)
	if !ok {
		return
	}
	path, line := payload.path, payload.line
	if path == "" || line <= 0 {
		return
	}
	v, err := e.OpenFile(path)
	if err != nil {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	text := doc.Text()
	lineIdx := line - 1
	lineStart, err := text.LineToChar(lineIdx)
	if err != nil {
		return
	}
	var lineEnd int
	if lineIdx+1 < text.LenLines() {
		lineEnd, err = text.LineToChar(lineIdx + 1)
		if err != nil {
			lineEnd = text.LenChars()
		}
	} else {
		lineEnd = text.LenChars()
	}
	sel, err := core.NewSelection(
		[]core.Range{core.NewRange(lineStart, lineEnd)}, 0,
	)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), sel)
}

func (gs *globalSearcher) scanLines(path string, scanner *bufio.Scanner) bool {
	rel, _ := filepath.Rel(gs.root, path)
	rel = filepath.ToSlash(rel)
	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++
		if !gs.re.MatchString(line) {
			continue
		}
		ln := lineNum
		select {
		case gs.ch <- PickerItem{
			Display: fmt.Sprintf("%s:%d", rel, ln),
			SortKey: fmt.Sprintf("%s:%06d", rel, ln),
			Location: PickerLocation{
				Target: PickerTarget{Path: path},
				Lines:  &PickerLineRange{From: ln - 1, To: ln - 1},
			},
			Payload: globalSearchPayload{path: path, line: ln},
		}:
		case <-gs.done:
			return false
		}
	}
	return true
}

func (gs *globalSearcher) searchFile(path string) bool {
	for _, snap := range gs.openDocs {
		if snap.path == path {
			return gs.scanLines(
				path, bufio.NewScanner(strings.NewReader(snap.text)),
			)
		}
	}
	f, err := os.Open(path)
	if err != nil {
		return true
	}
	defer func() { _ = f.Close() }()
	header := make([]byte, 1024)
	n, _ := f.Read(header)
	if looksBinary(header[:n]) {
		return true
	}
	if _, err := f.Seek(0, 0); err != nil {
		return true
	}
	return gs.scanLines(path, bufio.NewScanner(f))
}

func (gs *globalSearcher) walk() {
	defer close(gs.ch)
	_ = filepath.WalkDir(gs.root, func(
		path string, d os.DirEntry, err error,
	) error {
		if err != nil || path == gs.root {
			return nil
		}
		rel, err := filepath.Rel(gs.root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if skipPickerPath(rel, "", d, loadIgnoreFiles(gs.root, rel)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !gs.searchFile(path) {
			return filepath.SkipAll
		}
		return nil
	})
}

func globalSearchQuery(
	root string, openDocs []docSnap, pattern string, smartCase bool,
) (<-chan PickerItem, StopFunc) {
	ch := make(chan PickerItem, 256)
	rePattern := pattern
	if smartCase && !patternHasUpper(pattern) {
		rePattern = "(?i)" + pattern
	}
	re, err := regexp.Compile(rePattern)
	if err != nil {
		close(ch)
		return ch, func() {}
	}
	done := make(chan struct{})
	var once sync.Once
	cancel := func() { once.Do(func() { close(done) }) }
	gs := &globalSearcher{
		ch: ch, done: done, re: re, root: root, openDocs: openDocs,
	}
	go gs.walk()
	return ch, cancel
}

func patternHasUpper(s string) bool {
	for _, ch := range s {
		if unicode.IsUpper(ch) {
			return true
		}
	}
	return false
}
