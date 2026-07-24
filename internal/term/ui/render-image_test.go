package ui_test

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/builtin"
	"github.com/kode4food/toe/internal/term/builtin/files"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"

	"github.com/stretchr/testify/assert"
)

const imageEvictionCount = 30

func TestImageRender(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	root := t.TempDir()
	path := writeRenderImage(t, root, 40, 20, color.RGBA{G: 255, A: 255})

	e := view.NewEditor(root)
	openRenderImagePane(t, e, path)
	m := ui.New(e, command.NewKeymaps())
	m2, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = m2.(ui.Model)
	var rawMsgs []string
	m, rawMsgs = collectModelRawMsgs(m, cmd)
	raw := strings.Join(rawMsgs, "")

	content := m.View().Content

	assert.True(t,
		strings.ContainsRune(content, tui.PlaceholderRune),
	)
	// status shows IMG and image dimensions, not text-oriented fields
	out := stripANSI(content)
	assert.Contains(t, out, "IMG")
	assert.Contains(t, out, "40×20")
	assert.NotContains(t, out, "UTF-8")
	assert.Contains(t, raw, ",p=")
	assert.NotContains(t, raw, "d=i")
	assert.Contains(t, raw, "a=T")

	matches := regexp.MustCompile(`(?:\x1b_G|,)i=(\d+)`).
		FindAllStringSubmatch(raw, -1)
	assert.NotEmpty(t, matches)
	id, err := strconv.ParseUint(matches[0][1], 10, 32)
	assert.NoError(t, err)
	assert.LessOrEqual(t, id, uint64(0xFFFFFF))
	for _, match := range matches {
		assert.Equal(t, matches[0][1], match[1])
	}
	fg := fmt.Sprintf(
		"\x1b[38;2;%d;%d;%dm", id>>16, id>>8&0xFF, id&0xFF,
	)
	assert.Contains(t, content, fg)
}

func TestImageLoading(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	root := t.TempDir()
	path := writeRenderImage(t, root, 40, 20, nil)

	e := view.NewEditor(root)
	openRenderImagePane(t, e, path)
	m := ui.New(e, command.NewKeymaps())
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = m2.(ui.Model)

	out := stripANSI(m.View().Content)

	assert.Contains(t, out, i18n.Text(i18n.StatusImageLoading))
}

func TestImageUnsupported(t *testing.T) {
	for _, k := range []string{
		"KITTY_WINDOW_ID", "TERM", "TERM_PROGRAM", "KONSOLE_VERSION",
	} {
		t.Setenv(k, "")
	}
	root := t.TempDir()
	path := writeRenderImage(t, root, 40, 20, nil)

	e := view.NewEditor(root)
	openRenderImagePane(t, e, path)
	m := ui.New(e, command.NewKeymaps())
	m2, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = m2.(ui.Model)

	out := stripANSI(m.View().Content)
	assert.Contains(t, out, i18n.Text(i18n.StatusImageUnsupported))
	assert.False(t, strings.ContainsRune(out, tui.PlaceholderRune))

	_, rawMsgs := collectModelRawMsgs(m, cmd)
	assert.NotContains(t, strings.Join(rawMsgs, ""), "\x1b_G")
}

func TestOpenImagePane(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	root := t.TempDir()
	path := writeRenderImage(t, root, 40, 20, nil)
	e := view.NewEditor(root)

	v, ok, err := ui.OpenPath(e, path, ui.PickerAcceptReplace)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Nil(t, v)
	_, ok = e.Tree().Get(e.Tree().Focus()).(*ui.ImagePane)
	assert.True(t, ok)
}

func TestImageInput(t *testing.T) {
	root := t.TempDir()
	path := writeRenderImage(t, root, 40, 20, nil)
	e := view.NewEditor(root)
	e.Options().Mouse = true
	docID := e.Tree().Focus()
	km := command.NewKeymaps()
	m := ui.New(e, km)
	_, err := builtin.Register(m, km)
	assert.NoError(t, err)
	m = resize(m, 80, 24)
	pane, err := ui.NewImagePane(e, path)
	assert.NoError(t, err)
	assert.True(t, e.SplitPane(pane, view.LayoutVertical))
	e.FocusPane(docID)

	a := pane.Area()
	e.FocusPane(pane.ID())
	assert.Equal(t, view.ModeImage, e.Mode())

	// feed each update so the transmit lands and the paced wheel is not dropped
	feed := func(msg tea.Msg) {
		m2, cmd := m.Update(msg)
		m = feedImageMsgs(m2.(ui.Model), cmd)
	}
	feed(tea.KeyPressMsg{Code: '=', Text: "="})
	assert.Equal(t, 125, pane.Zoom())
	feed(tea.KeyPressMsg{Code: '-', Text: "-"})
	assert.Equal(t, 100, pane.Zoom())
	feed(tea.KeyPressMsg{Code: '+', Text: "+"})
	assert.Equal(t, 125, pane.Zoom())
	feed(tea.KeyPressMsg{Code: '0', Text: "0"})
	assert.Equal(t, 100, pane.Zoom())

	// a modified wheel zooms without stealing focus from the document
	e.FocusPane(docID)
	feed(tea.MouseWheelMsg{
		X: a.X + 1, Y: a.Y, Button: tea.MouseWheelUp, Mod: tea.ModCtrl,
	})
	assert.Equal(t, docID, e.Tree().Focus())
	assert.Equal(t, 110, pane.Zoom())
	feed(tea.MouseWheelMsg{
		X: a.X + 1, Y: a.Y, Button: tea.MouseWheelDown, Mod: tea.ModCtrl,
	})
	assert.Equal(t, 100, pane.Zoom())
}

// readyImage drives a transmit/ready cycle at the current zoom, then renders so
// the pan bounds are established for the panning tests
func readyImage(m ui.Model) ui.Model {
	m2, cmd := m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	m = feedImageMsgs(m2.(ui.Model), cmd)
	_ = m.View()
	return m
}

func TestImagePan(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	root := t.TempDir()
	path := writeRenderImage(t, root, 40, 20, nil)
	e := view.NewEditor(root)
	e.Options().Mouse = true
	openRenderImagePane(t, e, path)
	km := command.NewKeymaps()
	m := ui.New(e, km)
	_, err := builtin.Register(m, km)
	assert.NoError(t, err)
	m2, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = feedImageMsgs(m2.(ui.Model), cmd)
	pane, ok := e.FocusedPane().(*ui.ImagePane)
	assert.True(t, ok)

	// zoom past the pane so the grid overflows and can be panned
	for range 12 {
		pane.ZoomIn()
	}
	assert.Equal(t, 400, pane.Zoom())
	m = readyImage(m) // transmit + render at the zoomed size

	m = sendSpecialText(m, 'l', "l")
	right := pane.Pan().X
	assert.Positive(t, right)
	m = sendSpecialText(m, 'h', "h")
	assert.Equal(t, 0, pane.Pan().X)
	m = sendSpecialText(m, 'j', "j")
	assert.Positive(t, pane.Pan().Y)
	m = sendSpecialText(m, 'k', "k")
	assert.Equal(t, 0, pane.Pan().Y)

	// panning past the edge clamps rather than running away
	for range 200 {
		m = sendSpecialText(m, 'l', "l")
	}
	clamped := pane.Pan().X
	sendSpecialText(m, 'l', "l")
	assert.Equal(t, clamped, pane.Pan().X)
	assert.Greater(t, clamped, right)
}

func TestImageWheelDefault(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	root := t.TempDir()
	path := writeRenderImage(t, root, 40, 20, nil)
	e := view.NewEditor(root)
	e.Options().Mouse = true
	openRenderImagePane(t, e, path)
	km := command.NewKeymaps()
	m := ui.New(e, km)
	_, err := builtin.Register(m, km)
	assert.NoError(t, err)
	m2, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = feedImageMsgs(m2.(ui.Model), cmd)
	pane, ok := e.FocusedPane().(*ui.ImagePane)
	assert.True(t, ok)
	for range 12 {
		pane.ZoomIn()
	}
	m = readyImage(m)
	a := pane.Area()

	// a bare wheel pans; a modified wheel zooms
	m = mouse(m, tea.MouseWheelMsg{
		X: a.X + 1, Y: a.Y, Button: tea.MouseWheelRight,
	})
	assert.Positive(t, pane.Pan().X)
	assert.Equal(t, 400, pane.Zoom())
	mouse(m, tea.MouseWheelMsg{
		X: a.X + 1, Y: a.Y, Button: tea.MouseWheelDown, Mod: tea.ModCtrl,
	})
	assert.Equal(t, 390, pane.Zoom())
}

func TestImageWheelGestureLatch(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	root := t.TempDir()
	path := writeRenderImage(t, root, 40, 20, nil)
	e := view.NewEditor(root)
	e.Options().Mouse = true
	openRenderImagePane(t, e, path)
	km := command.NewKeymaps()
	m := ui.New(e, km)
	_, err := builtin.Register(m, km)
	assert.NoError(t, err)
	m2, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = feedImageMsgs(m2.(ui.Model), cmd)
	pane, ok := e.FocusedPane().(*ui.ImagePane)
	assert.True(t, ok)
	for range 12 {
		pane.ZoomIn()
	}
	m = readyImage(m)
	a := pane.Area()
	feed := func(msg tea.Msg) {
		m2, cmd := m.Update(msg)
		m = feedImageMsgs(m2.(ui.Model), cmd)
	}

	// a modified wheel starts a zoom gesture
	feed(tea.MouseWheelMsg{
		X: a.X + 1, Y: a.Y, Button: tea.MouseWheelDown, Mod: tea.ModCtrl,
	})
	assert.Equal(t, 390, pane.Zoom())

	// a bare wheel in the same stream stays a zoom, not a pan
	feed(tea.MouseWheelMsg{X: a.X + 1, Y: a.Y, Button: tea.MouseWheelDown})
	assert.Equal(t, 380, pane.Zoom())
	assert.Equal(t, 0, pane.Pan().Y)

	// after the gesture gap a bare wheel pans again
	time.Sleep(220 * time.Millisecond)
	feed(tea.MouseWheelMsg{X: a.X + 1, Y: a.Y, Button: tea.MouseWheelDown})
	assert.Equal(t, 380, pane.Zoom())
	assert.Positive(t, pane.Pan().Y)
}

func TestImageClickZoom(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	root := t.TempDir()
	path := writeRenderImage(t, root, 40, 20, nil)
	e := view.NewEditor(root)
	e.Options().Mouse = true
	docID := e.Tree().Focus()
	km := command.NewKeymaps()
	m := ui.New(e, km)
	_, err := builtin.Register(m, km)
	assert.NoError(t, err)
	m = resize(m, 80, 24)
	pane, err := ui.NewImagePane(e, path)
	assert.NoError(t, err)
	assert.True(t, e.SplitPane(pane, view.LayoutVertical))
	e.FocusPane(docID)
	a := pane.Area()

	// a click zooms in
	m = mouse(m, tea.MouseClickMsg{X: a.X + 1, Y: a.Y, Button: tea.MouseLeft})
	assert.Equal(t, pane.ID(), e.Tree().Focus())
	assert.Equal(t, 125, pane.Zoom())

	// a modified click zooms out
	mouse(m, tea.MouseClickMsg{
		X: a.X + 1, Y: a.Y, Button: tea.MouseLeft, Mod: tea.ModCtrl,
	})
	assert.Equal(t, 100, pane.Zoom())
}

func TestImagePanRestore(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	root := t.TempDir()
	path := writeRenderImage(t, root, 40, 20, nil)
	session := filepath.Join(root, "session.toml")

	// session 1: zoom and pan through the model, then save
	e := view.NewEditor(root)
	e.Options().Mouse = true
	openRenderImagePane(t, e, path)
	km := command.NewKeymaps()
	m := ui.New(e, km)
	_, err := builtin.Register(m, km)
	assert.NoError(t, err)
	m2, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = feedImageMsgs(m2.(ui.Model), cmd)
	pane, ok := e.FocusedPane().(*ui.ImagePane)
	assert.True(t, ok)
	for range 12 {
		pane.ZoomIn()
	}
	m = readyImage(m)
	m = sendSpecialText(m, 'l', "l")
	sendSpecialText(m, 'j', "j")
	want := pane.Pan()
	assert.NotEqual(t, geom.Point{}, want)
	assert.NoError(t, e.SaveSession(session, nil))

	// session 2: faithful startup — restore before the first WindowSize
	next := view.NewEditor(root)
	next.Options().Mouse = true
	nm := ui.New(next, command.NewKeymaps())
	_, restored, err := next.RestoreSession(session)
	assert.NoError(t, err)
	assert.True(t, restored)
	nm2, ncmd := nm.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	nm = feedImageMsgs(nm2.(ui.Model), ncmd)
	_ = nm.View()
	restoredPane, ok := next.FocusedPane().(*ui.ImagePane)
	assert.True(t, ok)
	assert.Equal(t, want, restoredPane.Pan())
}

// A render at a smaller bound (a shrunk window, or a layout transient during
// restore) must not rewrite the stored pan: only a zoom converges it, so the
// offset survives a resize round-trip
func TestImagePanSurvivesResize(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	root := t.TempDir()
	path := writeRenderImage(t, root, 40, 20, nil)
	e := view.NewEditor(root)
	e.Options().Mouse = true
	openRenderImagePane(t, e, path)
	km := command.NewKeymaps()
	m := ui.New(e, km)
	_, err := builtin.Register(m, km)
	assert.NoError(t, err)
	m2, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = feedImageMsgs(m2.(ui.Model), cmd)
	pane, ok := e.FocusedPane().(*ui.ImagePane)
	assert.True(t, ok)
	for range 12 {
		pane.ZoomIn()
	}
	m = readyImage(m)
	for range 40 {
		m = sendSpecialText(m, 'l', "l")
		m = sendSpecialText(m, 'j', "j")
	}
	want := pane.Pan()
	assert.NotEqual(t, geom.Point{}, want)

	// shrink the window (bound gets smaller), render, then restore the size
	m2, cmd = m.Update(tea.WindowSizeMsg{Width: 30, Height: 10})
	m = feedImageMsgs(m2.(ui.Model), cmd)
	_ = m.View()
	m2, cmd = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = feedImageMsgs(m2.(ui.Model), cmd)
	_ = m.View()

	assert.Equal(t, want, pane.Pan())
}

// Zooming out shrinks the overflow, so the pan converges to center; once fitted
// there is no pan left, and zooming back in starts from center, not the old
// offset
func TestImagePanForgottenOnZoomOut(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	root := t.TempDir()
	path := writeRenderImage(t, root, 40, 20, nil)
	e := view.NewEditor(root)
	e.Options().Mouse = true
	openRenderImagePane(t, e, path)
	km := command.NewKeymaps()
	m := ui.New(e, km)
	_, err := builtin.Register(m, km)
	assert.NoError(t, err)
	m2, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = feedImageMsgs(m2.(ui.Model), cmd)
	pane, ok := e.FocusedPane().(*ui.ImagePane)
	assert.True(t, ok)
	for range 12 {
		pane.ZoomIn()
	}
	m = readyImage(m)
	m = sendSpecialText(m, 'l', "l")
	m = sendSpecialText(m, 'j', "j")
	assert.NotEqual(t, geom.Point{}, pane.Pan())

	// zoom out to a fit: the pan converges to center
	for range 12 {
		pane.ZoomOut()
	}
	assert.Equal(t, 100, pane.Zoom())
	m = readyImage(m)
	assert.Equal(t, geom.Point{}, pane.Pan())

	// zoom back in: still centered, the old offset is gone
	for range 12 {
		pane.ZoomIn()
	}
	readyImage(m)
	assert.Equal(t, geom.Point{}, pane.Pan())
}

func walkCmdMsgs(cmd tea.Cmd, fn func(tea.Msg)) {
	if cmd == nil {
		return
	}
	msg := cmd()
	if msg == nil {
		return
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			walkCmdMsgs(c, fn)
		}
		return
	}
	rv := reflect.ValueOf(msg)
	if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Func {
		for i := 0; i < rv.Len(); i++ {
			if c, ok := rv.Index(i).Interface().(tea.Cmd); ok {
				walkCmdMsgs(c, fn)
			}
		}
		return
	}
	fn(msg)
}

func collectModelRawMsgs(m ui.Model, cmd tea.Cmd) (ui.Model, []string) {
	var out []string
	walkCmdMsgs(cmd, func(msg tea.Msg) {
		if raw, ok := msg.(tea.RawMsg); ok {
			out = append(out, fmt.Sprint(raw.Msg))
			return
		}
		m2, next := m.Update(msg)
		m = m2.(ui.Model)
		var more []string
		m, more = collectModelRawMsgs(m, next)
		out = append(out, more...)
	})
	return m, out
}

// feedImageMsgs feeds every message a cmd produces back through Update, which
// delivers the imageReadyMsg that ungates placeholder rendering
func feedImageMsgs(m ui.Model, cmd tea.Cmd) ui.Model {
	m, _ = collectModelRawMsgs(m, cmd)
	return m
}

func TestImagePickerPreviewTransmit(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	dir := t.TempDir()
	writeRenderImage(t, dir, 40, 20, color.RGBA{B: 255, A: 255})

	e := view.NewEditor(dir)
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "file_picker", m.PickerAction(files.NewFilePickerInDir(dir)),
		[]command.KeyEvent{char('p')},
	)
	m = resize(m, 120, 30)
	m2, cmd := m.Update(tea.KeyPressMsg{Code: 'p', Text: "p"})
	_, rawMsgs := collectModelRawMsgs(m2.(ui.Model), cmd)
	raw := strings.Join(rawMsgs, "")

	assert.Contains(t, raw, ",p=")
	assert.NotContains(t, raw, "d=i")
	assert.Contains(t, raw, "\x1b_Gf=100")
	assert.Contains(t, raw, "U=1")
}

func TestImagePickerPreviewLoading(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	dir := t.TempDir()
	writeRenderImage(t, dir, 40, 20, color.RGBA{B: 255, A: 255})

	e := view.NewEditor(dir)
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "file_picker", m.PickerAction(files.NewFilePickerInDir(dir)),
		[]command.KeyEvent{char('p')},
	)
	m = resize(m, 120, 30)
	m2, _ := m.Update(tea.KeyPressMsg{Code: 'p', Text: "p"})
	m = m2.(ui.Model)

	out := stripANSI(m.View().Content)

	assert.Contains(t, out, i18n.Text(i18n.StatusImageLoading))
}

func TestImagePickerPreview(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	tmp := t.TempDir()
	writeRenderImage(t, tmp, 40, 20, color.RGBA{B: 255, A: 255})

	e := view.NewEditor(tmp)
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "file_picker", m.PickerAction(files.NewFilePickerInDir(tmp)),
		[]command.KeyEvent{char('p')},
	)
	m = resize(m, 120, 30)
	m2, cmd := m.Update(tea.KeyPressMsg{Code: 'p', Text: "p"})
	m = feedImageMsgs(m2.(ui.Model), cmd)

	assert.True(t,
		strings.ContainsRune(m.View().Content, tui.PlaceholderRune),
	)
}

func TestImagePickerPreviewResize(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	tmp := t.TempDir()
	writeRenderImage(t, tmp, 40, 20, color.RGBA{B: 255, A: 255})

	e := view.NewEditor(tmp)
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "file_picker", m.PickerAction(files.NewFilePickerInDir(tmp)),
		[]command.KeyEvent{char('p')},
	)
	m = resize(m, 120, 30)
	m2, cmd := m.Update(tea.KeyPressMsg{Code: 'p', Text: "p"})
	m = feedImageMsgs(m2.(ui.Model), cmd)

	assert.NotPanics(t, func() {
		_ = m.View()
		m2, _ = m.Update(tea.MouseClickMsg{
			X: 60, Y: 8, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		m2, cmd = m.Update(tea.MouseMotionMsg{
			X: 75, Y: 8, Button: tea.MouseLeft,
		})
		m, _ = collectModelRawMsgs(m2.(ui.Model), cmd)
		m2, _ = m.Update(tea.MouseReleaseMsg{
			Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		_ = m.View()
	})
}

func TestImageSplit(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	root := t.TempDir()
	path := writeRenderImage(t, root, 40, 20, nil)

	e := view.NewEditor(root)
	openRenderImagePane(t, e, path)
	pane, err := ui.NewImagePane(e, path)
	assert.NoError(t, err)
	e.Tree().Split(pane, view.LayoutVertical)
	m := ui.New(e, command.NewKeymaps())
	_, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	_, rawMsgs := collectModelRawMsgs(m, cmd)
	raw := strings.Join(rawMsgs, "")

	assert.Equal(t, 2, strings.Count(raw, "f=100"))
	assert.Equal(t, 2, strings.Count(raw, "U=1"))
}

func TestImageRestore(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	root := t.TempDir()
	path := writeRenderImage(t, root, 40, 20, nil)
	session := filepath.Join(root, "session.toml")
	e := view.NewEditor(root)
	e.ResizeTree(geom.Size{Width: 80, Height: 24})
	openRenderImagePane(t, e, path)
	pane, err := ui.NewImagePane(e, path)
	assert.NoError(t, err)
	e.Tree().Split(pane, view.LayoutVertical)
	assert.NoError(t, e.SaveSession(session, nil))

	next := view.NewEditor(root)
	m := ui.New(next, command.NewKeymaps()) // registers the image pane factory
	_, restored, err := next.RestoreSession(session)
	assert.NoError(t, err)
	assert.True(t, restored)
	m2, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, rawMsgs := collectModelRawMsgs(m2.(ui.Model), cmd)
	raw := strings.Join(rawMsgs, "")

	assert.Equal(t, 2, strings.Count(raw, ",p="))
	assert.Equal(t, 2, strings.Count(raw, "\x1b_Gf=100"))
	assert.True(t, strings.ContainsRune(m.View().Content, tui.PlaceholderRune))
}

func TestImageResizeOrder(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	root := t.TempDir()
	path := writeRenderImage(t, root, 40, 20, nil)
	e := view.NewEditor(root)
	openRenderImagePane(t, e, path)
	m := ui.New(e, command.NewKeymaps())
	m2, firstCmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = m2.(ui.Model)
	m2, resizeCmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m, resizeRaw := collectModelRawMsgs(m2.(ui.Model), resizeCmd)
	m, firstRaw := collectModelRawMsgs(m, firstCmd)
	all := strings.Join(resizeRaw, "") + strings.Join(firstRaw, "")

	// a resize before the first transmit lands still re-places at the live size
	assert.Equal(t, 1, strings.Count(all, "a=T"))
	assert.Contains(t, all, "c=100,r=25")
	assert.True(t, strings.ContainsRune(m.View().Content, tui.PlaceholderRune))
}

func TestImageZoomKeepsReadyPlacement(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	root := t.TempDir()
	path := writeRenderImage(t, root, 40, 20, nil)
	e := view.NewEditor(root)
	openRenderImagePane(t, e, path)
	m := ui.New(e, command.NewKeymaps())
	m2, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = feedImageMsgs(m2.(ui.Model), cmd)

	pane, ok := e.FocusedPane().(*ui.ImagePane)
	assert.True(t, ok)
	pane.ZoomIn()
	pane.MarkDirty()
	m2, _ = m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	m = m2.(ui.Model)

	content := m.View().Content

	assert.True(t, strings.ContainsRune(content, tui.PlaceholderRune))
	assert.NotContains(t,
		stripANSI(content), i18n.Text(i18n.StatusImageLoading),
	)
}

func TestImageZoomPending(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	root := t.TempDir()
	path := writeRenderImage(t, root, 40, 20, nil)
	e := view.NewEditor(root)
	openRenderImagePane(t, e, path)
	m := ui.New(e, command.NewKeymaps())
	m2, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, initialRaw := collectModelRawMsgs(m2.(ui.Model), cmd)

	pane, ok := e.FocusedPane().(*ui.ImagePane)
	assert.True(t, ok)
	pane.ZoomOut()
	m2, firstCmd := m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	m = m2.(ui.Model)
	pending := m.View().Content
	m2, duplicateCmd := m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	m, duplicateRaw := collectModelRawMsgs(m2.(ui.Model), duplicateCmd)
	m, firstRaw := collectModelRawMsgs(m, firstCmd)

	raw := strings.Join(firstRaw, "")
	assert.Empty(t, duplicateRaw)
	assert.Equal(t, 1, strings.Count(raw, "a=p"))

	re := regexp.MustCompile(`(?:\x1b_G|,)p=(\d+)`)
	initial := re.FindStringSubmatch(strings.Join(initialRaw, ""))
	next := re.FindStringSubmatch(raw)
	if !assert.NotEmpty(t, initial) || !assert.NotEmpty(t, next) {
		return
	}
	assert.NotEqual(t, initial[1], next[1])

	initialID, err := strconv.ParseUint(initial[1], 10, 32)
	assert.NoError(t, err)
	nextID, err := strconv.ParseUint(next[1], 10, 32)
	assert.NoError(t, err)
	initialColor := fmt.Sprintf(
		"\x1b[58:2::%d:%d:%dm",
		initialID>>16, initialID>>8&0xFF, initialID&0xFF,
	)
	nextColor := fmt.Sprintf(
		"\x1b[58:2::%d:%d:%dm",
		nextID>>16, nextID>>8&0xFF, nextID&0xFF,
	)
	assert.Contains(t, pending, initialColor)
	assert.NotContains(t, pending, nextColor)
	assert.Contains(t, m.View().Content, nextColor)
}

func TestImageEviction(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_TTY", "")
	dir := t.TempDir()
	for i := range imageEvictionCount {
		writeDistinctImage(t, dir, i)
	}

	e := view.NewEditor(dir)
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "file_picker", m.PickerAction(files.NewFilePickerInDir(dir)),
		[]command.KeyEvent{char('p')},
	)
	m = resize(m, 120, 30)
	m2, cmd := m.Update(tea.KeyPressMsg{Code: 'p', Text: "p"})
	m, raw := collectModelRawMsgs(m2.(ui.Model), cmd)

	var all strings.Builder
	all.WriteString(strings.Join(raw, ""))
	for range imageEvictionCount {
		m2, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		var more []string
		m, more = collectModelRawMsgs(m2.(ui.Model), cmd)
		all.WriteString(strings.Join(more, ""))
	}

	assert.Contains(t, all.String(), "a=d")
	assert.Contains(t, all.String(), "d=I")
}

func writeDistinctImage(t testing.TB, dir string, i int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 40, 20))
	img.Set(0, 0, color.RGBA{R: uint8(i), G: uint8(i * 7), B: 255, A: 255})
	var buf bytes.Buffer
	assert.NoError(t, png.Encode(&buf, img))
	path := filepath.Join(dir, fmt.Sprintf("pic%02d.png", i))
	assert.NoError(t, os.WriteFile(path, buf.Bytes(), 0o644))
}

func writeRenderImage(
	t testing.TB, dir string, w, h int, c color.Color,
) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	if c != nil {
		img.Set(0, 0, c)
	}
	var buf bytes.Buffer
	assert.NoError(t, png.Encode(&buf, img))
	path := filepath.Join(dir, "pic.png")
	assert.NoError(t, os.WriteFile(path, buf.Bytes(), 0o644))
	return path
}

func openRenderImagePane(t testing.TB, e *view.Editor, path string) {
	t.Helper()
	pane, err := ui.NewImagePane(e, path)
	assert.NoError(t, err)
	e.ReplacePane(e.Tree().Focus(), pane)
}
