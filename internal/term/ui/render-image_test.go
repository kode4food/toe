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
	assert.Contains(t, raw, "d=i,a=d")
	assert.NotContains(t, raw, "d=I")
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
	m2, _ := m.Update(tea.MouseClickMsg{
		X: a.X + 1, Y: a.Y, Button: tea.MouseLeft,
	})
	m = m2.(ui.Model)
	assert.Equal(t, pane.ID(), e.Tree().Focus())
	assert.Equal(t, view.ModeImage, e.Mode())

	m2, _ = m.Update(tea.KeyPressMsg{Code: '=', Text: "="})
	m = m2.(ui.Model)
	assert.Equal(t, 125, pane.Zoom())
	m2, _ = m.Update(tea.KeyPressMsg{Code: '-', Text: "-"})
	m = m2.(ui.Model)
	assert.Equal(t, 100, pane.Zoom())
	m2, _ = m.Update(tea.KeyPressMsg{Code: '+', Text: "+"})
	m = m2.(ui.Model)
	assert.Equal(t, 125, pane.Zoom())
	_, _ = m.Update(tea.KeyPressMsg{Code: '0', Text: "0"})
	assert.Equal(t, 100, pane.Zoom())

	e.FocusPane(docID)
	m2, _ = m.Update(tea.MouseWheelMsg{
		X: a.X + 1, Y: a.Y, Button: tea.MouseWheelUp,
	})
	m = m2.(ui.Model)
	assert.Equal(t, pane.ID(), e.Tree().Focus())
	assert.Equal(t, 125, pane.Zoom())
	_, _ = m.Update(tea.MouseWheelMsg{
		X: a.X + 1, Y: a.Y, Button: tea.MouseWheelDown,
	})
	assert.Equal(t, 100, pane.Zoom())
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

	assert.Contains(t, raw, "d=i,a=d")
	assert.NotContains(t, raw, "d=I")
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

	assert.Equal(t, 2, strings.Count(raw, "d=i,a=d"))
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
	m2, oldCmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = m2.(ui.Model)
	m2, newCmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m, rawMsgs := collectModelRawMsgs(m2.(ui.Model), newCmd)
	raw := strings.Join(rawMsgs, "")
	m, rawMsgs = collectModelRawMsgs(m, oldCmd)
	oldRaw := strings.Join(rawMsgs, "")

	assert.Contains(t, raw, "d=i,a=d")
	assert.Empty(t, oldRaw)
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
