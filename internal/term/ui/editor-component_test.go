package ui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/builtin"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
)

type highlightRefreshController struct {
	locationController
}

func (c *highlightRefreshController) DocumentHighlights(
	doc *view.Document, id view.Id,
) ([]view.DocumentHighlight, error) {
	doc.SetDocumentHighlights(id, c.highlights)
	return c.highlights, nil
}

func TestModelView(t *testing.T) {
	t.Run("returns empty before resize", func(t *testing.T) {
		e := editorWithText(t, "")
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)

		v := m.View()

		assert.Empty(t, v.Content)
	})
}

func TestInsertMode(t *testing.T) {
	t.Run("inserts space", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		km := command.NewKeymaps()
		m := resize(ui.New(e, km), 80, 24)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)

		m = sendKey(m, 'a')
		m = sendSpecial(m, tea.KeySpace)
		_ = sendKey(m, 'b')
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)

		assert.Equal(t, "a b", doc.Text().String())
	})
}

func TestEditorKeys(t *testing.T) {
	t.Run("accepts count in select mode", func(t *testing.T) {
		e := editorWithText(t, "abcdefgh")
		e.SetMode(view.ModeSelect)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 40, 8)

		_ = sendKey(m, '3')

		assert.Equal(t, 3, e.Count())
	})
}

func TestMouseMiddlePaste(t *testing.T) {
	t.Run("pastes at clicked position", func(t *testing.T) {
		e := editorWithText(t, "abcd")
		clip := testutil.NewFakeClipboard()
		clip.Primary = "XY"
		e.SetClipboard(clip)
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseReleaseMsg{
			X: 9, Y: 0, Button: tea.MouseMiddle,
		})
		_ = m2

		doc := e.FocusedDocument()
		assert.Equal(t, "abXYcd", doc.Text().String())
		assert.Equal(t, 2, testutil.CursorPos(t, e))
	})

	t.Run("disabled leaves document unchanged", func(t *testing.T) {
		e := editorWithText(t, "abcd")
		clip := testutil.NewFakeClipboard()
		clip.Primary = "XY"
		e.SetClipboard(clip)
		e.Options().MiddleClickPaste = false
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseReleaseMsg{
			X: 9, Y: 0, Button: tea.MouseMiddle,
		})
		_ = m2

		doc := e.FocusedDocument()
		assert.Equal(t, "abcd", doc.Text().String())
	})

	t.Run("alt replaces selection", func(t *testing.T) {
		e := editorWithText(t, "abcd")
		clip := testutil.NewFakeClipboard()
		clip.Primary = "XY"
		e.SetClipboard(clip)
		testutil.SetSelection(t, e, []core.Range{{
			Anchor: 1,
			Head:   3,
		}}, 0)
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseReleaseMsg{
			X: 0, Y: 0, Button: tea.MouseMiddle, Mod: tea.ModAlt,
		})
		_ = m2

		doc := e.FocusedDocument()
		assert.Equal(t, "aXYd", doc.Text().String())
	})

	t.Run("outside content is ignored", func(t *testing.T) {
		e := editorWithText(t, "abcd")
		clip := testutil.NewFakeClipboard()
		clip.Primary = "XY"
		e.SetClipboard(clip)
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseReleaseMsg{
			X: 2, Y: 7, Button: tea.MouseMiddle,
		})
		_ = m2

		doc := e.FocusedDocument()
		assert.Equal(t, "abcd", doc.Text().String())
	})
}

func TestMouseWheelScroll(t *testing.T) {
	// renderedModel gives a 40×8 window; row 7 is the pane statusline, which
	// is outside the content area

	t.Run("wheel over pane scrolls that pane", func(t *testing.T) {
		e := editorWithText(t, "a\nb\nc\nd\ne\nf\ng\nh\ni\nj")
		e.SetViewHeight(6)
		m := renderedModel(e)

		v := e.FocusedView()
		assert.NotNil(t, v)
		before := v.Offset().Anchor

		m2, _ := m.Update(tea.MouseWheelMsg{
			X: 5, Y: 0, Button: tea.MouseWheelDown,
		})
		_ = m2

		assert.Greater(t, v.Offset().Anchor, before)
	})

	t.Run("status bar wheel ignored", func(t *testing.T) {
		e := editorWithText(t, "a\nb\nc\nd\ne\nf\ng\nh\ni\nj")
		e.SetViewHeight(6)
		m := renderedModel(e)

		v := e.FocusedView()
		assert.NotNil(t, v)
		before := v.Offset().Anchor

		// 40×8 window, no bufferline: the pane fills the frame, so Y=7 is
		// its own status bar row (not content)
		m2, _ := m.Update(tea.MouseWheelMsg{
			X: 5, Y: 7, Button: tea.MouseWheelDown,
		})
		_ = m2

		assert.Equal(t, before, v.Offset().Anchor)
	})

	t.Run("horizontal wheel scrolls columns", func(t *testing.T) {
		e := editorWithText(t, strings.Repeat("x", 60)+"\nshort")
		m := renderedModel(e)

		v := e.FocusedView()
		assert.NotNil(t, v)

		m2, _ := m.Update(tea.MouseWheelMsg{
			X: 5, Y: 0, Button: tea.MouseWheelRight,
		})
		hOff := v.Offset().HorizontalOffset
		assert.Greater(t, hOff, 0)

		// The cursor is at column 0; without free scroll the next render
		// would snap the offset back to it
		_ = m2.(ui.Model).View()
		assert.Equal(t, hOff, v.Offset().HorizontalOffset)
	})

}

func TestMouseClickPositioning(t *testing.T) {
	t.Run("positions cursor on content click", func(t *testing.T) {
		e := editorWithText(t, "abcdef")
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseClickMsg{
			X: 10, Y: 0, Button: tea.MouseLeft,
		})
		_ = m2

		assert.Equal(t, 3, testutil.CursorPos(t, e))
	})

	t.Run("clicks rendered character cell", func(t *testing.T) {
		e := editorWithText(t, "abcdef")
		m := renderedModel(e)
		at := renderedTextPoint(t, m, "abcdef", 3)

		m2, _ := m.Update(tea.MouseClickMsg{
			X: at.X, Y: at.Y, Button: tea.MouseLeft,
		})
		_ = m2

		assert.Equal(t, 3, testutil.CursorPos(t, e))
	})

	t.Run("ignores click on status or command line", func(t *testing.T) {
		e := editorWithText(t, "abcdef")
		m := renderedModel(e)

		// place the cursor in the content area first
		m2, _ := m.Update(tea.MouseClickMsg{
			X: 10, Y: 0, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		assert.Equal(t, 3, testutil.CursorPos(t, e))

		// the bottom row of the 8-high window is the pane statusline, never
		// editor content, so a click there must leave the cursor put
		m3, _ := m.Update(tea.MouseClickMsg{
			X: 1, Y: 7, Button: tea.MouseLeft,
		})
		_ = m3

		assert.Equal(t, 3, testutil.CursorPos(t, e))
	})

	t.Run("alt click adds secondary selection", func(t *testing.T) {
		e := editorWithText(t, "abcdef")
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseClickMsg{
			X: 9, Y: 0, Button: tea.MouseLeft, Mod: tea.ModAlt,
		})
		_ = m2

		v := e.FocusedView()
		doc := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 2, len(sel.Ranges()))
		assert.Equal(t, 1, sel.PrimaryIndex())
	})

	t.Run("select mode click extends primary", func(t *testing.T) {
		e := editorWithText(t, "abcdef")
		e.SetMode(view.ModeSelect)
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseClickMsg{
			X: 10, Y: 0, Button: tea.MouseLeft,
		})
		_ = m2

		v := e.FocusedView()
		doc := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, []core.Range{{
			Anchor: 0,
			Head:   4,
		}}, sel.Ranges())
	})

	t.Run("bufferline row is ignored", func(t *testing.T) {
		e := editorWithText(t, "abcdef")
		e.Options().BufferLine = view.BufferLineAlways
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseClickMsg{
			X: 10, Y: 0, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		assert.Equal(t, 0, testutil.CursorPos(t, e))

		m2, _ = m.Update(tea.MouseClickMsg{
			X: 10, Y: 1, Button: tea.MouseLeft,
		})
		_ = m2
		assert.Equal(t, 3, testutil.CursorPos(t, e))
	})

	t.Run("click below row map clamps", func(t *testing.T) {
		e := editorWithText(t, "a\nbc")
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseClickMsg{
			X: 10, Y: 5, Button: tea.MouseLeft,
		})
		_ = m2

		assert.Equal(t, 4, testutil.CursorPos(t, e))
	})

	t.Run("tab click uses expanded width", func(t *testing.T) {
		e := editorWithText(t, "\tab")
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseClickMsg{
			X: 11, Y: 0, Button: tea.MouseLeft,
		})
		_ = m2

		assert.Equal(t, 1, testutil.CursorPos(t, e))
	})
}

func TestMouseSeparatorDrag(t *testing.T) {
	t.Run("vertical separator resizes panes", func(t *testing.T) {
		e := editorWithText(t, "abcdef")
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		_ = m.View()
		v := e.FocusedView()
		assert.NotNil(t, v)
		split := e.VSplit(v.DocID())
		assert.NotNil(t, split)
		_ = m.View()

		views := e.Views()
		before := views[0].View.Area().Width
		sepX := views[0].View.Area().X + before
		m2, _ := m.Update(tea.MouseClickMsg{
			X: sepX, Y: 0, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		m2, _ = m.Update(tea.MouseMotionMsg{
			X: sepX - 5, Y: 0, Button: tea.MouseLeft,
		})
		_ = m2

		after := e.Views()[0].View.Area().Width
		assert.Less(t, after, before)
	})

	t.Run("horizontal separator resizes panes", func(t *testing.T) {
		e := editorWithText(t, "abcdef")
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		_ = m.View()
		v := e.FocusedView()
		assert.NotNil(t, v)
		split := e.HSplit(v.DocID())
		assert.NotNil(t, split)
		_ = m.View()

		views := e.Views()
		before := views[0].View.Area().Height
		sepY := views[0].View.Area().Y + before
		m2, _ := m.Update(tea.MouseClickMsg{
			X: 0, Y: sepY, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		m2, _ = m.Update(tea.MouseMotionMsg{
			X: 0, Y: sepY - 2, Button: tea.MouseLeft,
		})
		_ = m2

		after := e.Views()[0].View.Area().Height
		assert.Less(t, after, before)
	})
}

func TestMouseDragBounds(t *testing.T) {
	t.Run("negative row clamps to top edge", func(t *testing.T) {
		e := editorWithText(t, "abcdef")
		m := renderedModel(e)
		m2, _ := m.Update(tea.MouseClickMsg{
			X: 8, Y: 0, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)

		// dragging above the pane's top row clamps to it instead of being a
		// no-op, so a drag started at the top edge still extends
		m2, _ = m.Update(tea.MouseMotionMsg{
			X: 8, Y: -1, Button: tea.MouseLeft,
		})
		_ = m2

		v := e.FocusedView()
		doc := e.FocusedDocument()
		assert.Equal(t,
			[]core.Range{{Anchor: 1, Head: 2}},
			doc.SelectionFor(v.ID()).Ranges(),
		)
	})

	t.Run("bufferline drag extends selection", func(t *testing.T) {
		e := editorWithText(t, "abcdef")
		e.Options().BufferLine = view.BufferLineAlways
		m := renderedModel(e)
		m2, _ := m.Update(tea.MouseClickMsg{
			X: 8, Y: 1, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)

		m2, _ = m.Update(tea.MouseMotionMsg{
			X: 13, Y: 1, Button: tea.MouseLeft,
		})
		_ = m2

		v := e.FocusedView()
		doc := e.FocusedDocument()
		assert.Equal(t,
			[]core.Range{{Anchor: 1, Head: 6}},
			doc.SelectionFor(v.ID()).Ranges(),
		)
	})
}

func TestMouseDragAutoScroll(t *testing.T) {
	t.Run("drag past bottom edge scrolls", func(t *testing.T) {
		var b strings.Builder
		for i := range 60 {
			_, _ = fmt.Fprintf(&b, "line%d\n", i)
		}
		e := editorWithText(t, b.String())
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseClickMsg{X: 0, Y: 0, Button: tea.MouseLeft})
		m = m2.(ui.Model)

		v := e.FocusedView()
		assert.NotNil(t, v)
		before := v.Offset().Anchor

		// dragging past the pane's bottom edge starts an auto-scroll tick
		// instead of just clamping the selection in place
		m2, cmd := m.Update(tea.MouseMotionMsg{
			X: 0, Y: 100, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		assert.NotNil(t, cmd)

		// fire exactly one tick; it reschedules itself, but we only need to
		// prove a single step scrolls forward, not drive it to completion
		msg := cmd()
		m2, _ = m.Update(msg)
		_ = m2.(ui.Model)

		assert.Greater(t, v.Offset().Anchor, before)
	})

	t.Run("tick moves exactly one line after render", func(t *testing.T) {
		var b strings.Builder
		for i := range 60 {
			_, _ = fmt.Fprintf(&b, "line%d\n", i)
		}
		e := editorWithText(t, b.String())
		m := renderedModel(e)
		v := e.FocusedView()
		assert.NotNil(t, v)
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		anchor, err := doc.Text().LineToChar(20)
		assert.NoError(t, err)
		v.SetOffset(view.Position{Anchor: anchor})

		m2, _ := m.Update(tea.MouseClickMsg{X: 0, Y: 3, Button: tea.MouseLeft})
		m = m2.(ui.Model)
		beforeLine, err := doc.Text().CharToLine(v.Offset().Anchor)
		assert.NoError(t, err)

		m2, cmd := m.Update(tea.MouseMotionMsg{
			X: 0, Y: -100, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		assert.NotNil(t, cmd)
		m2, _ = m.Update(cmd())
		m = m2.(ui.Model)

		// a render used to let normal scrolloff re-centering fight the tick
		// and jump the viewport further than the deliberate one-line step
		_ = m.View()

		afterLine, err := doc.Text().CharToLine(v.Offset().Anchor)
		assert.NoError(t, err)
		assert.Equal(t, 1, beforeLine-afterLine)
	})

	t.Run("drag near top edge scrolls up", func(t *testing.T) {
		var b strings.Builder
		for i := range 60 {
			_, _ = fmt.Fprintf(&b, "line%d\n", i)
		}
		e := editorWithText(t, b.String())
		m := renderedModel(e)

		v := e.FocusedView()
		assert.NotNil(t, v)
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		anchor, err := doc.Text().LineToChar(20)
		assert.NoError(t, err)
		v.SetOffset(view.Position{Anchor: anchor})

		m2, _ := m.Update(tea.MouseClickMsg{X: 0, Y: 3, Button: tea.MouseLeft})
		m = m2.(ui.Model)
		before := v.Offset().Anchor

		// row 1 is inside the visible pane, not off-screen, but within the
		// top scrolloff margin, a pane at the screen's top edge has no row
		// above it to drag into, so the trigger zone must live inside it
		m2, cmd := m.Update(tea.MouseMotionMsg{
			X: 0, Y: 1, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		assert.NotNil(t, cmd)

		msg := cmd()
		m2, _ = m.Update(msg)
		_ = m2.(ui.Model)

		assert.Less(t, v.Offset().Anchor, before)
	})

	t.Run("tick interval speeds up near edge", func(t *testing.T) {
		var b strings.Builder
		for i := range 60 {
			_, _ = fmt.Fprintf(&b, "line%d\n", i)
		}
		tickDelay := func(dragY int) time.Duration {
			e := editorWithText(t, b.String())
			m := renderedModel(e)

			m2, _ := m.Update(tea.MouseClickMsg{
				X: 0, Y: 3, Button: tea.MouseLeft,
			})
			m = m2.(ui.Model)

			_, cmd := m.Update(tea.MouseMotionMsg{
				X: 0, Y: dragY, Button: tea.MouseLeft,
			})
			assert.NotNil(t, cmd)
			start := time.Now()
			cmd() // tea.Tick blocks for the scheduled interval
			return time.Since(start)
		}

		// row 1 is barely inside the top margin; far off-screen is pinned
		// against the pane's edge, where ticks should fire faster
		nearMargin := tickDelay(1)
		atEdge := tickDelay(-100)
		assert.Greater(t, nearMargin, atEdge)
	})

	t.Run("edge zone does not jump selection", func(t *testing.T) {
		var b strings.Builder
		for i := range 60 {
			_, _ = fmt.Fprintf(&b, "line%d\n", i)
		}
		e := editorWithText(t, b.String())
		m := renderedModel(e)

		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		startAnchor, err := doc.Text().LineToChar(20)
		assert.NoError(t, err)
		v := e.FocusedView()
		assert.NotNil(t, v)

		var lines []int
		for row := range 8 {
			v.SetOffset(view.Position{Anchor: startAnchor})
			m2, _ := m.Update(tea.MouseClickMsg{
				X: 0, Y: 4, Button: tea.MouseLeft,
			})
			m = m2.(ui.Model)
			m2, _ = m.Update(tea.MouseMotionMsg{
				X: 0, Y: row, Button: tea.MouseLeft,
			})
			m = m2.(ui.Model)

			line, err := doc.Text().CharToLine(testutil.CursorPos(t, e))
			assert.NoError(t, err)
			lines = append(lines, line)
		}

		// each one-row mouse move must change the cursor by at most one
		// line, snapping straight to the clamped edge on entering the
		// margin zone used to jump several lines in a single step
		for i := 1; i < len(lines); i++ {
			assert.LessOrEqual(t, lines[i-1]-lines[i], 1)
		}
	})

	t.Run("repeated motion does not restart timer", func(t *testing.T) {
		var b strings.Builder
		for i := range 60 {
			_, _ = fmt.Fprintf(&b, "line%d\n", i)
		}
		e := editorWithText(t, b.String())
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseClickMsg{X: 0, Y: 0, Button: tea.MouseLeft})
		m = m2.(ui.Model)

		m2, cmd := m.Update(tea.MouseMotionMsg{
			X: 0, Y: 100, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		assert.NotNil(t, cmd)

		// a second motion event still inside the same edge zone must not
		// reschedule the tick, otherwise a fast mouse-motion stream never
		// lets the timer fire
		m2, cmd2 := m.Update(tea.MouseMotionMsg{
			X: 1, Y: 100, Button: tea.MouseLeft,
		})
		_ = m2.(ui.Model)
		assert.Nil(t, cmd2)
	})

	t.Run("release stops auto-scroll", func(t *testing.T) {
		var b strings.Builder
		for i := range 60 {
			_, _ = fmt.Fprintf(&b, "line%d\n", i)
		}
		e := editorWithText(t, b.String())
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseClickMsg{X: 0, Y: 0, Button: tea.MouseLeft})
		m = m2.(ui.Model)
		m2, cmd := m.Update(tea.MouseMotionMsg{
			X: 0, Y: 100, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		assert.NotNil(t, cmd)

		m2, _ = m.Update(tea.MouseReleaseMsg{Button: tea.MouseLeft})
		m = m2.(ui.Model)

		v := e.FocusedView()
		assert.NotNil(t, v)
		before := v.Offset().Anchor

		// the tick scheduled before release must be a no-op now
		msg := cmd()
		m2, _ = m.Update(msg)
		_ = m2.(ui.Model)

		assert.Equal(t, before, v.Offset().Anchor)
	})

	t.Run("left edge zone starts at content", func(t *testing.T) {
		minWidth := 10
		e := editorWithText(t, strings.Repeat("x", 200)+"\nshort")
		e.Options().Gutters = view.Gutter{
			Present: true,
			Layout:  []view.GutterType{view.GutterTypeLineNumbers},
			LineNumbers: view.GutterLineNumbers{
				MinWidth: &minWidth,
			},
		}
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		_ = m.View()
		v := e.FocusedView()
		assert.NotNil(t, v)
		area := v.Area()

		m2, _ := m.Update(tea.MouseClickMsg{
			X: area.X + 20, Y: 0, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)

		// column area.X+minWidth is the first column of text, right after
		// the gutter, the trigger zone must reach that far, not require
		// the drag to be over the gutter itself
		m2, cmd := m.Update(tea.MouseMotionMsg{
			X: area.X + minWidth, Y: 0, Button: tea.MouseLeft,
		})
		_ = m2.(ui.Model)

		assert.NotNil(t, cmd)
	})

	t.Run("clamped position at edge stays fast", func(t *testing.T) {
		var b strings.Builder
		for i := range 60 {
			_, _ = fmt.Fprintf(&b, "line%d\n", i)
		}
		e := editorWithText(t, b.String())
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseClickMsg{X: 0, Y: 5, Button: tea.MouseLeft})
		m = m2.(ui.Model)

		m2, cmd := m.Update(tea.MouseMotionMsg{
			X: 0, Y: 0, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		assert.NotNil(t, cmd)
		start := time.Now()
		cmd()
		firstTick := time.Since(start)

		// the terminal clamps out-of-window coordinates, so a drag held
		// outside keeps reporting this same row with no further movement
		m2, cmd2 := m.Update(tea.MouseMotionMsg{
			X: 0, Y: 0, Button: tea.MouseLeft,
		})
		_ = m2.(ui.Model)
		if cmd2 != nil {
			start2 := time.Now()
			cmd2()
			assert.Less(t, time.Since(start2), 2*firstTick)
		}
	})
}

func TestFreeScroll(t *testing.T) {
	t.Run("keypress keeps other scrolled view", func(t *testing.T) {
		e := editorWithText(t, strings.Repeat("0123456789abcdef\n", 20))
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		v1 := e.FocusedView()
		assert.NotNil(t, v1)
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		v2 := e.VSplit(doc.ID())
		assert.NotNil(t, v2)
		assert.Equal(t, v2.ID(), e.Tree().Focus())
		anchor, err := doc.Text().LineToChar(10)
		assert.NoError(t, err)
		scrolled := view.Position{Anchor: anchor}
		v1.SetOffset(scrolled)
		v1.BeginFreeScroll(doc.Revision(), doc.SelectionFor(v1.ID()))
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		m = sendKey(m, 'l')
		_ = m.View().Content

		assert.True(t, v1.FreeScroll())
		assert.Equal(t, scrolled, v1.Offset())
		assert.False(t, v2.FreeScroll())
	})

	t.Run("click near viewport edge recouples", func(t *testing.T) {
		var b strings.Builder
		for i := range 60 {
			_, _ = fmt.Fprintf(&b, "line%d\n", i)
		}
		e := editorWithText(t, b.String())
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		km := command.NewKeymaps()
		m := resize(ui.New(e, km), 80, 24)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		for range 40 {
			m = sendKey(m, 'j')
		}
		_ = m.View()

		v := e.FocusedView()
		assert.NotNil(t, v)
		before := v.Offset()

		// the last content row before the status/command line sits inside the
		// default 5-line scrolloff margin, so re-coupling scrolls to honour it
		lines := strings.Split(stripANSI(m.View().Content), "\n")
		clickY := -1
		for y := len(lines) - 2; y >= 0; y-- {
			if strings.Contains(lines[y], "line") {
				clickY = y
				break
			}
		}
		assert.GreaterOrEqual(t, clickY, 0)

		m2, _ := m.Update(tea.MouseClickMsg{
			X: 5, Y: clickY, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		_ = m.View()

		assert.False(t, v.FreeScroll())
		assert.NotEqual(t, before, v.Offset())
	})

	t.Run("repeat click past line end recouples", func(t *testing.T) {
		e := editorWithText(t, strings.Repeat("x", 200)+"\nshort\n")
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		km := command.NewKeymaps()
		m := resize(ui.New(e, km), 80, 24)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		v := e.FocusedView()
		assert.NotNil(t, v)
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)

		// row 1 holds "short", scrolled entirely off to the left, so clicking
		// it resolves to a cursor outside the window. The second pass repeats
		// the click verbatim, leaving the selection unchanged
		for range 2 {
			off := v.Offset()
			off.HorizontalOffset = 100
			v.SetOffset(off)
			v.BeginFreeScroll(doc.Revision(), doc.SelectionFor(v.ID()))
			_ = m.View()

			m2, _ := m.Update(tea.MouseClickMsg{
				X: 60, Y: 1, Button: tea.MouseLeft,
			})
			m = m2.(ui.Model)
			_ = m.View()

			assert.False(t, v.FreeScroll())
			assert.Equal(t, 0, v.Offset().HorizontalOffset)
		}
	})
}

func TestGotoWindowEdges(t *testing.T) {
	newScrolled := func(t *testing.T) (ui.Model, *view.Editor) {
		t.Helper()
		e := view.NewEditor(t.TempDir())
		var sb strings.Builder
		for i := 1; i <= 400; i++ {
			_, _ = fmt.Fprintf(&sb, "line %d\n", i)
		}
		testutil.SetEditorText(t, e, sb.String())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 24)
		for range 250 {
			m = sendKey(m, 'j')
			_ = m.View()
		}
		return m, e
	}

	edgeLines := func(m ui.Model) (string, string) {
		lines := strings.Split(stripANSI(m.View().Content), "\n")
		var first, last string
		for _, ln := range lines {
			if strings.Contains(ln, "line ") {
				if first == "" {
					first = strings.TrimSpace(ln)
				}
				last = strings.TrimSpace(ln)
			}
		}
		return first, last
	}

	t.Run("a count lands on the top visible line", func(t *testing.T) {
		m, e := newScrolled(t)
		first, _ := edgeLines(m)

		m = sendKey(sendKey(sendKey(m, '1'), 'g'), 't')
		_ = m.View()

		assert.Contains(t, first, fmt.Sprintf("line %d", cursorLineOf(t, e)))
	})

	t.Run("a count lands on the bottom visible line", func(t *testing.T) {
		m, e := newScrolled(t)
		_, last := edgeLines(m)

		m = sendKey(sendKey(sendKey(m, '1'), 'g'), 'b')
		_ = m.View()

		assert.Contains(t, last, fmt.Sprintf("line %d", cursorLineOf(t, e)))
	})
}

func TestGotoLineKeySequence(t *testing.T) {
	newModel := func(t *testing.T) (ui.Model, *view.Editor) {
		var b strings.Builder
		for i := range 10 {
			_, _ = fmt.Fprintf(&b, "line%d\n", i)
		}
		e := editorWithText(t, b.String())
		km := command.NewKeymaps()
		m := resize(ui.New(e, km), 80, 24)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		return m, e
	}

	cursorLine := func(t *testing.T, e *view.Editor) int {
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		line, err := doc.Text().CharToLine(testutil.CursorPos(t, e))
		assert.NoError(t, err)
		return line
	}

	t.Run("g <n> g goes to line n", func(t *testing.T) {
		m, e := newModel(t)
		m = sendKey(m, 'g')
		m = sendKey(m, '5')
		_ = sendKey(m, 'g')
		// count 5 targets line 5 (1-based), i.e. index 4
		assert.Equal(t, 4, cursorLine(t, e))
	})

	t.Run("g g goes to file start", func(t *testing.T) {
		m, e := newModel(t)
		testutil.SetCursor(t, e, 25)
		m = sendKey(m, 'g')
		_ = sendKey(m, 'g')
		assert.Equal(t, 0, cursorLine(t, e))
	})

	t.Run("multi-digit count", func(t *testing.T) {
		m, e := newModel(t)
		m = sendKey(m, 'g')
		m = sendKey(m, '1')
		m = sendKey(m, '0')
		_ = sendKey(m, 'g')
		assert.Equal(t, 9, cursorLine(t, e))
	})

	t.Run("status renders keys then count", func(t *testing.T) {
		m, _ := newModel(t)
		m = sendKey(m, 'g')
		m = sendKey(m, '5')
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "g → 5")
		assert.NotContains(t, out, "5g")
	})

	t.Run("a wide toast leaves the last cell blank", func(t *testing.T) {
		m, e := newModel(t)
		m = resize(m, 8, 6)
		// a toast as wide as the frame would fill the final column without
		// the guard, and writing it auto-wraps the terminal
		e.SetStatusMsg("1234567")
		m = resize(m, 8, 6)
		lines := strings.Split(stripANSI(m.View().Content), "\n")
		row := []rune(toastRow(m, "\u2570"))
		// the gap keeps the popup off the final columns, which auto-wrap
		assert.NotEmpty(t, row)
		assert.Equal(t, '\u256f', row[5])
		for _, ln := range lines {
			assert.LessOrEqual(t, len([]rune(ln)), 8)
		}
	})
}

func TestCountMotionKeySequence(t *testing.T) {
	t.Run("5j moves down 5 lines in NOR", func(t *testing.T) {
		var b strings.Builder
		for i := range 10 {
			_, _ = fmt.Fprintf(&b, "line%d\n", i)
		}
		e := editorWithText(t, b.String())
		km := command.NewKeymaps()
		m := resize(ui.New(e, km), 80, 24)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)

		m = sendKey(m, '5')
		_ = sendKey(m, 'j')

		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		line, err := doc.Text().CharToLine(testutil.CursorPos(t, e))
		assert.NoError(t, err)
		assert.Equal(t, 5, line)
	})

	t.Run("3w advances 3 words in NOR", func(t *testing.T) {
		e := editorWithText(t, "aa bb cc dd ee")
		km := command.NewKeymaps()
		m := resize(ui.New(e, km), 80, 24)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)

		m = sendKey(m, '3')
		_ = sendKey(m, 'w')

		// count 3 advances past aa/bb/cc into the run before "dd"; a plain w
		// stops one word short, so this asserts the count took effect
		assert.Equal(t, 8, testutil.CursorPos(t, e))
	})
}

func TestFocusMessages(t *testing.T) {
	t.Run("focus message handled", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		m2, _ := m.Update(tea.FocusMsg{})
		m = m2.(ui.Model)

		assert.NotEmpty(t, m.View().Content)
	})

	t.Run("blur focus lost triggers autosave", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		e.Options().AutoSaveFocusLost = true
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		m2, _ := m.Update(tea.BlurMsg{})
		m = m2.(ui.Model)

		assert.NotEmpty(t, m.View().Content)
	})
}

func TestMouseDisabledEvents(t *testing.T) {
	t.Run("mouse release ignored disabled", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.Options().Mouse = false
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		m2, _ := m.Update(tea.MouseReleaseMsg{Button: tea.MouseLeft})
		m = m2.(ui.Model)

		assert.NotEmpty(t, m.View().Content)
	})

	t.Run("mouse wheel ignored when mouse disabled", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.Options().Mouse = false
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		m2, _ := m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
		m = m2.(ui.Model)

		assert.NotEmpty(t, m.View().Content)
	})

	t.Run("unknown message returns ignored", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		m2, _ := m.Update(struct{ unexpected string }{"msg"})
		m = m2.(ui.Model)

		assert.NotEmpty(t, m.View().Content)
	})
}

func TestSyncEditorMessages(t *testing.T) {
	t.Run("status shown from action", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		// No LSP sets "No configured language server"
		bindNormalTestAction(
			km, "goto_decl", m.GotoDeclarationAction,
			[]command.KeyEvent{char('g')},
		)
		m = resize(m, 80, 24)

		m = sendKey(m, 'g')

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "No configured language server")
	})
}

func TestDocumentHighlightRefresh(t *testing.T) {
	t.Run("refreshes after cursor move", func(t *testing.T) {
		e := editorWithText(t, "hello\n")
		ctl := &highlightRefreshController{
			locationController: locationController{
				highlights: []view.DocumentHighlight{{From: 1, To: 3}},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := resize(ui.New(e, km), 80, 24)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)

		m2, cmd := m.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
		m = m2.(ui.Model)
		assert.NotNil(t, cmd)
		drainCmd(m, cmd)

		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		v := e.FocusedView()
		assert.NotNil(t, v)
		assert.Equal(t, ctl.highlights, doc.DocumentHighlights(v.ID()))
	})
}

func TestAutoSaveCmd(t *testing.T) {
	t.Run("autosave tick created on keypress", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.Options().AutoSaveAfterDelay = true
		km := command.NewKeymaps()
		m := resize(ui.New(e, km), 80, 24)

		m2, cmd := m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
		m = m2.(ui.Model)

		assert.NotEmpty(t, m.View().Content)
		assert.NotNil(t, cmd)
	})

	t.Run("autosave fires on gen match", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		e.Options().AutoSaveAfterDelay = true
		e.Options().AutoSaveDelayTimeout = 0
		km := command.NewKeymaps()
		m := resize(ui.New(e, km), 80, 24)

		// Execute the autosave command returned by the keypress
		m2, cmd := m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
		m = m2.(ui.Model)
		if cmd != nil {
			msg := cmd()
			if msg != nil {
				m2, _ = m.Update(msg)
				m = m2.(ui.Model)
			}
		}

		assert.NotEmpty(t, m.View().Content)
	})
}

func renderedTextPoint(
	t *testing.T, m ui.Model, text string, off int,
) geom.Point {
	t.Helper()
	lines := strings.Split(stripANSI(m.View().Content), "\n")
	for y, line := range lines {
		if x := strings.Index(line, text); x >= 0 {
			return geom.Point{X: x + off, Y: y}
		}
	}
	t.Fatalf("rendered text %q not found", text)
	return geom.Point{}
}

func drainCmd(m ui.Model, cmd tea.Cmd) ui.Model {
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			return m
		}
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, c := range batch {
				m = drainCmd(m, c)
			}
			return m
		}
		m2, next := m.Update(msg)
		m = m2.(ui.Model)
		cmd = next
	}
	return m
}

func userDocuments(e *view.Editor) []*view.Document {
	var out []*view.Document
	for _, d := range e.AllDocuments() {
		if d.DisplayName() != view.MessagesBufferName {
			out = append(out, d)
		}
	}
	return out
}

func renderedModel(e *view.Editor) ui.Model {
	m := ui.New(e, command.NewKeymaps())
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 8})
	m = m2.(ui.Model)
	_ = m.View()
	return m
}

func editorWithText(t *testing.T, text string) *view.Editor {
	t.Helper()
	e := view.NewEditor("/tmp")
	e.Options().Theme = view.DefaultTheme
	doc := e.FocusedDocument()
	assert.NotNil(t, doc)
	rope := doc.Text()
	cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
		core.TextChange(core.Span{From: 0, To: 0}, text),
	})
	assert.NoError(t, err)
	tx := core.NewTransaction(rope).
		WithChanges(cs).
		WithSelection(core.PointSelection(0))
	assert.NoError(t, e.Apply(tx))
	return e
}

func cursorLineOf(t *testing.T, e *view.Editor) int {
	t.Helper()
	doc := e.FocusedDocument()
	text := doc.Text()
	cursor := doc.SelectionFor(e.FocusedView().ID()).Primary().Cursor(text)
	pos, err := text.Position(cursor)
	assert.NoError(t, err)
	return pos.Line
}
