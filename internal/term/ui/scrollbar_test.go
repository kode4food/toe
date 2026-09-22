package ui_test

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"

	"github.com/stretchr/testify/assert"
)

type scrollbarCell struct {
	glyph rune
	style styledRuneStyle
}

const (
	scrollbarWidth  = 40
	scrollbarHeight = 12
	scrollbarRows   = scrollbarHeight - 1 // the status line takes the last row

	scrollbarThumbBg = "48;2;61;62;81"    // base tinted toward overlay0
	scrollbarErrorFg = "38;2;243;139;168" // mocha red
)

var (
	scrollbarThinGlyphs  = []rune{'▔', '─', '▁'}
	scrollbarBlockGlyphs = []rune{'▀', '▄', '█'}
)

func TestScrollbar(t *testing.T) {
	t.Run("hidden unless enabled", func(t *testing.T) {
		cells := scrollbarCells(t,
			scrollbarEditor(t, 200, false), scrollbarRows,
		)

		for _, cell := range cells {
			assert.NotEqual(t, scrollbarThumbBg, cell.style.bg)
		}
	})

	t.Run("draws a track and a thumb", func(t *testing.T) {
		cells := scrollbarCells(t, scrollbarEditor(t, 200, true), scrollbarRows)

		assert.Len(t, cells, scrollbarRows)
		assert.Equal(t, []int{0}, thumbRows(cells))
	})

	t.Run("thumb fills when nothing scrolls", func(t *testing.T) {
		cells := scrollbarCells(t, scrollbarEditor(t, 0, true), scrollbarRows)

		assert.Len(t, thumbRows(cells), scrollbarRows)
	})

	t.Run("thumb follows the scroll position", func(t *testing.T) {
		e := editorWithText(t, numberedLines(200))
		e.Options().Scrollbar = true
		m := resize(
			ui.New(e, command.NewKeymaps()), scrollbarWidth, scrollbarHeight,
		)
		_ = m.View().Content

		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		last, err := doc.Text().LineToChar(doc.Text().LenLines() - 1)
		assert.NoError(t, err)
		assert.NoError(t, e.Apply(core.NewTransaction(doc.Text()).
			WithSelection(core.PointSelection(last)),
		))

		cells := scrollbarCells(t, m.View().Content, scrollbarRows)

		assert.Equal(t, []int{scrollbarRows - 1}, thumbRows(cells))
	})

	t.Run("marks a single diagnostic with a rule", func(t *testing.T) {
		e := editorWithText(t, numberedLines(200))
		e.Options().Scrollbar = true
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		at, err := doc.Text().LineToChar(180)
		assert.NoError(t, err)
		doc.ReplaceDiagnostics("test", []view.Diagnostic{{
			Range:    core.Span{From: at, To: at + 1},
			Severity: view.DiagnosticSeverityError,
		}})
		m := resize(
			ui.New(e, command.NewKeymaps()), scrollbarWidth, scrollbarHeight,
		)

		cells := scrollbarCells(t, m.View().Content, scrollbarRows)

		marked := markRows(cells)
		assert.Len(t, marked, 2)
		row := marked[1]
		assert.Contains(t, scrollbarThinGlyphs, cells[row].glyph)
		assert.Equal(t, scrollbarErrorFg, cells[row].style.fg)
	})

	t.Run("a run of lines draws one half block", func(t *testing.T) {
		e := editorWithText(t, numberedLines(200))
		e.Options().Scrollbar = true
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		var diags []view.Diagnostic
		for _, line := range []int{150, 151, 152} {
			at, err := doc.Text().LineToChar(line)
			assert.NoError(t, err)
			diags = append(diags, view.Diagnostic{
				Range:    core.Span{From: at, To: at + 1},
				Severity: view.DiagnosticSeverityError,
			})
		}
		doc.ReplaceDiagnostics("test", diags)
		m := resize(
			ui.New(e, command.NewKeymaps()), scrollbarWidth, scrollbarHeight,
		)

		cells := scrollbarCells(t, m.View().Content, scrollbarRows)

		marked := markRows(cells)
		assert.Len(t, marked, 2)
		row := marked[1]
		assert.Contains(t, scrollbarBlockGlyphs, cells[row].glyph)
		assert.Equal(t, scrollbarErrorFg, cells[row].style.fg)
	})

	t.Run("marks the cursor line", func(t *testing.T) {
		cells := scrollbarCells(t, scrollbarEditor(t, 200, true), scrollbarRows)

		assert.Equal(t, []int{0}, markRows(cells))
		assert.Contains(t, scrollbarThinGlyphs, cells[0].glyph)
	})

	t.Run("marks search matches", func(t *testing.T) {
		e := editorWithText(t, numberedLines(200))
		e.Options().Scrollbar = true
		v := e.FocusedView()
		assert.NotNil(t, v)
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		doc.ShowSearchHighlights(v.ID())
		e.Registers().Set('/', "line 150")
		m := resize(
			ui.New(e, command.NewKeymaps()), scrollbarWidth, scrollbarHeight,
		)

		cells := scrollbarCells(t, m.View().Content, scrollbarRows)

		marked := markRows(cells)
		assert.Len(t, marked, 2)
		assert.Contains(t, scrollbarThinGlyphs, cells[marked[1]].glyph)
	})

	t.Run("clicking the bar scrolls there", func(t *testing.T) {
		e := editorWithText(t, numberedLines(200))
		e.Options().Scrollbar = true
		m := resize(
			ui.New(e, command.NewKeymaps()), scrollbarWidth, scrollbarHeight,
		)
		_ = m.View().Content

		m = mouse(m, tea.MouseClickMsg{
			X: scrollbarWidth - 1, Y: scrollbarRows - 1, Button: tea.MouseLeft,
		})

		cells := scrollbarCells(t, m.View().Content, scrollbarRows)
		assert.Greater(t, topVisibleLine(t, m.View().Content), 150)
		assert.Contains(t, thumbRows(cells), scrollbarRows-1)
	})

	t.Run("dragging the bar keeps scrolling", func(t *testing.T) {
		e := editorWithText(t, numberedLines(200))
		e.Options().Scrollbar = true
		m := resize(
			ui.New(e, command.NewKeymaps()), scrollbarWidth, scrollbarHeight,
		)
		_ = m.View().Content

		m = mouse(m, tea.MouseClickMsg{
			X: scrollbarWidth - 1, Y: scrollbarRows - 1, Button: tea.MouseLeft,
		})
		m = mouse(m, tea.MouseMotionMsg{
			X: scrollbarWidth - 1, Y: 0, Button: tea.MouseLeft,
		})

		assert.Contains(t, firstLine(m.View().Content), "line 0")
	})

	t.Run("clicking the bar leaves the cursor alone", func(t *testing.T) {
		e := editorWithText(t, numberedLines(200))
		e.Options().Scrollbar = true
		m := resize(
			ui.New(e, command.NewKeymaps()), scrollbarWidth, scrollbarHeight,
		)
		_ = m.View().Content

		before := cursorLineOf(t, e)
		mouse(m, tea.MouseClickMsg{
			X: scrollbarWidth - 1, Y: 4, Button: tea.MouseLeft,
		})

		assert.Equal(t, before, cursorLineOf(t, e))
	})

	t.Run("thumb holds every visible line's mark", func(t *testing.T) {
		e := editorWithText(t, numberedLines(200))
		e.Options().Scrollbar = true
		m := resize(
			ui.New(e, command.NewKeymaps()), scrollbarWidth, scrollbarHeight,
		)
		_ = m.View().Content

		for _, row := range []int{0, 3, 5, 8, scrollbarRows - 1} {
			click := tea.MouseClickMsg{
				X: scrollbarWidth - 1, Y: row, Button: tea.MouseLeft,
			}
			m = mouse(m, click)
			clean := scrollbarCells(t, m.View().Content, scrollbarRows)
			top := topVisibleLine(t, m.View().Content)
			putCursorOnLine(t, e, top)
			markEveryLine(t, e, top, top+scrollbarRows-1)
			m = mouse(m, click)

			cells := scrollbarCells(t, m.View().Content, scrollbarRows)

			assert.NotEmpty(t, markRows(cells))
			assert.Subset(t, thumbRows(clean), markRows(cells))
		}
	})

	t.Run("clicking centers the thumb on the row", func(t *testing.T) {
		e := editorWithText(t, numberedLines(200))
		e.Options().Scrollbar = true
		m := resize(
			ui.New(e, command.NewKeymaps()), scrollbarWidth, scrollbarHeight,
		)
		_ = m.View().Content

		for _, row := range []int{0, 3, 5, 8, scrollbarRows - 1} {
			m = mouse(m, tea.MouseClickMsg{
				X: scrollbarWidth - 1, Y: row, Button: tea.MouseLeft,
			})

			cells := scrollbarCells(t, m.View().Content, scrollbarRows)

			assert.Contains(t, thumbRows(cells), row)
		}
	})

	t.Run("narrows the text area by one column", func(t *testing.T) {
		wide := strings.Repeat("x", 200) + "\n"
		e := editorWithText(t, wide)
		e.Options().Scrollbar = false
		m := resize(
			ui.New(e, command.NewKeymaps()), scrollbarWidth, scrollbarHeight,
		)
		before := strings.Count(firstLine(m.View().Content), "x")

		e.Options().Scrollbar = true
		e.Options().Gen++
		after := strings.Count(firstLine(m.View().Content), "x")

		assert.Equal(t, before-1, after)
	})
}

func thumbRows(cells []scrollbarCell) []int {
	var out []int
	for i, cell := range cells {
		if cell.style.bg == scrollbarThumbBg {
			out = append(out, i)
		}
	}
	return out
}

func markRows(cells []scrollbarCell) []int {
	var out []int
	for i, cell := range cells {
		if cell.glyph != ' ' {
			out = append(out, i)
		}
	}
	return out
}

func topVisibleLine(t *testing.T, content string) int {
	t.Helper()
	var line int
	_, err := fmt.Sscanf(strings.TrimSpace(firstLine(content)), "%d", &line)
	assert.NoError(t, err)
	return line - 1
}

func putCursorOnLine(t *testing.T, e *view.Editor, line int) {
	t.Helper()
	doc := e.FocusedDocument()
	assert.NotNil(t, doc)
	at, err := doc.Text().LineToChar(line)
	assert.NoError(t, err)
	assert.NoError(t, e.Apply(core.NewTransaction(doc.Text()).
		WithSelection(core.PointSelection(at)),
	))
}

func markEveryLine(t *testing.T, e *view.Editor, from, to int) {
	t.Helper()
	doc := e.FocusedDocument()
	assert.NotNil(t, doc)
	var diags []view.Diagnostic
	for line := from; line <= to && line < doc.Text().LenLines(); line++ {
		at, err := doc.Text().LineToChar(line)
		assert.NoError(t, err)
		diags = append(diags, view.Diagnostic{
			Provider: "test",
			Range:    core.Span{From: at, To: at + 1},
			Severity: view.DiagnosticSeverityError,
		})
	}
	doc.ReplaceDiagnostics("test", diags)
}

func scrollbarEditor(t *testing.T, lines int, bar bool) string {
	t.Helper()
	e := editorWithText(t, numberedLines(lines))
	e.Options().Scrollbar = bar
	m := resize(
		ui.New(e, command.NewKeymaps()), scrollbarWidth, scrollbarHeight,
	)
	return m.View().Content
}

func numberedLines(n int) string {
	var sb strings.Builder
	for i := range n {
		fmt.Fprintf(&sb, "line %d\n", i)
	}
	return sb.String()
}

func firstLine(content string) string {
	return strings.SplitN(stripANSI(content), "\n", 2)[0]
}

// scrollbarCells returns the rightmost cell of each content row
func scrollbarCells(t *testing.T, content string, count int) []scrollbarCell {
	t.Helper()
	rows := strings.Split(content, "\n")
	assert.GreaterOrEqual(t, len(rows), count)
	out := make([]scrollbarCell, 0, count)
	for _, row := range rows[:count] {
		cells := rowCells(row)
		out = append(out, cells[len(cells)-1])
	}
	return out
}

func rowCells(row string) []scrollbarCell {
	var cur styledRuneStyle
	var out []scrollbarCell
	for len(row) > 0 {
		if strings.HasPrefix(row, "\x1b[") {
			end := strings.IndexByte(row, 'm')
			if end < 0 {
				break
			}
			cur = updateStyle(cur, row[2:end])
			row = row[end+1:]
			continue
		}
		r, n := utf8.DecodeRuneInString(row)
		out = append(out, scrollbarCell{glyph: r, style: cur})
		row = row[n:]
	}
	return out
}
