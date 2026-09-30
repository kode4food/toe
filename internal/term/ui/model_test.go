package ui_test

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin/files"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

type retainedPickerSource struct {
	feedPickerSource
	loads int
}

const setUnicodeCore = "\x1b[?2027h"

// Load counts each scan of the source
func (r *retainedPickerSource) Load() ui.PickerLoad {
	r.loads++
	return r.feedPickerSource.Load()
}

func TestModelLifecycle(t *testing.T) {
	newModel := func() ui.Model {
		e := view.NewEditor(t.TempDir())
		return resize(ui.New(e, command.NewKeymaps()), 80, 24)
	}

	t.Run("init produces a renderable model", func(t *testing.T) {
		m := newModel()
		_ = m.Init()
		assert.NotEmpty(t, m.View().Content)
	})

	t.Run("view before resize is empty", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		m := ui.New(e, command.NewKeymaps())
		v := m.View()

		assert.Equal(t, "", v.Content)
		assert.True(t, v.AltScreen)
	})

	t.Run("with startup cmd renders", func(t *testing.T) {
		m := newModel().WithStartupCmd(nil)
		assert.NotEmpty(t, m.View().Content)
	})

	t.Run("startup message appears in view", func(t *testing.T) {
		m := settled(newModel().WithStartupMessage("hello startup"))
		assert.Contains(t, stripANSI(m.View().Content), "hello startup")
	})

	t.Run("initial picker mounts on resize", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		m := ui.New(e, command.NewKeymaps()).
			WithInitialPicker(files.NewFilePicker)
		m = resize(m, 80, 24)
		assert.NotEmpty(t, m.View().Content)
	})

	t.Run("initial nil picker is ignored", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		m := ui.New(e, command.NewKeymaps()).WithInitialPicker(
			func(*view.Editor) *ui.Picker { return nil },
		)
		m = resize(m, 80, 24)
		assert.NotEmpty(t, m.View().Content)
	})
}

func TestModelResume(t *testing.T) {
	resume := func(t *testing.T, reports ...tea.Msg) string {
		t.Helper()
		e := view.NewEditor(t.TempDir())
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		for _, r := range reports {
			m2, _ := m.Update(r)
			m = m2.(ui.Model)
		}
		m2, cmd := m.Update(tea.ResumeMsg{})
		_, raws := collectModelRawMsgs(m2.(ui.Model), cmd)
		return strings.Join(raws, "")
	}

	t.Run("restores grapheme clustering", func(t *testing.T) {
		raw := resume(t, tea.ModeReportMsg{
			Mode:  ansi.ModeUnicodeCore,
			Value: ansi.ModeReset,
		})
		assert.Contains(t, raw, setUnicodeCore)
	})

	t.Run("skips unsupported terminal", func(t *testing.T) {
		raw := resume(t, tea.ModeReportMsg{
			Mode:  ansi.ModeUnicodeCore,
			Value: ansi.ModeNotRecognized,
		})
		assert.NotContains(t, raw, setUnicodeCore)
	})

	t.Run("skips unasked terminal", func(t *testing.T) {
		assert.NotContains(t, resume(t), setUnicodeCore)
	})
}

func TestCommandPaletteAction(t *testing.T) {
	t.Run("opens command palette picker", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "palette", m.CommandPaletteAction,
			[]command.KeyEvent{char('f')},
		)
		m = resize(m, 80, 24)
		m = sendKey(m, 'f')
		out := stripANSI(m.View().Content)
		assert.NotEmpty(t, out)
	})
}

func TestLastPickerAction(t *testing.T) {
	t.Run("no last picker is noop", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "last", m.LastPickerAction, []command.KeyEvent{char('l')},
		)
		m = resize(m, 80, 24)
		m = sendKey(m, 'l')
		assert.NotEmpty(t, m.View().Content)
	})

	t.Run("reopens last picker", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "palette", m.CommandPaletteAction,
			[]command.KeyEvent{char('f')},
		)
		bindNormalTestAction(
			km, "last", m.LastPickerAction, []command.KeyEvent{char('l')},
		)
		m = resize(m, 80, 24)
		m = sendKey(m, 'f')
		m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
		m = m2.(ui.Model)
		m = sendKey(m, 'l')
		assert.NotEmpty(t, m.View().Content)
	})
}

func TestPickerState(t *testing.T) {
	t.Run("scopes by starting point", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		start := "first"
		bind := func(name string, key rune, title string) {
			bindNormalTestAction(
				km, name, m.PickerAction(func(*view.Editor) *ui.Picker {
					return ui.NewPicker(e, fixedPickerSource{
						items:    fixedPickerItems(3),
						title:    title,
						stateKey: start,
					})
				}), []command.KeyEvent{char(key)},
			)
		}
		bind("first", 'p', "first-picker")
		bind("second", 'o', "second-picker")
		m = resize(m, 120, 24)

		m = typeQuery(sendKey(m, 'p'), "alpha")
		m = sendSpecial(m, tea.KeyDown)
		m = sendSpecial(m, tea.KeyDown)
		m = sendSpecial(m, tea.KeyEscape)
		m = typeQuery(sendKey(m, 'o'), "beta")
		m = sendSpecial(m, tea.KeyEscape)
		m = sendKey(m, 'p')
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "alpha")
		assert.Contains(t, out, "CONTENT-02")
		m = sendSpecial(m, tea.KeyEscape)

		start = "second"
		m = sendKey(m, 'p')
		assert.NotContains(t, stripANSI(m.View().Content), "alpha")
	})

	t.Run("reveals restored selection", func(t *testing.T) {
		paths := make([]string, 1200)
		for i := range paths {
			paths[i] = fmt.Sprintf("file-%04d", i)
		}
		m := feedPickerModel(t, paths)
		m = sendKeyAndFeed(m, 'p')
		m = sendSpecial(m, tea.KeyEnd)
		m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
		m = m2.(ui.Model)
		m2, cmd := m.Update(tea.KeyPressMsg{Code: 'p', Text: "p"})
		m = m2.(ui.Model)
		assert.NotPanics(t, func() { m.View() })
		assert.Contains(t, stripANSI(m.View().Content), "file-1199")
		assert.Contains(t, stripANSI(m.View().Content), "1200/1200")
		for cmd != nil {
			m2, cmd = m.Update(firstMsg(cmd))
			m = m2.(ui.Model)
			assert.NotPanics(t, func() { m.View() })
		}

		assert.Contains(t, stripANSI(m.View().Content), "file-1199")
	})

	t.Run("finishes while another picker is open", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		t.Cleanup(m.Close)
		src := &retainedPickerSource{
			Ident: "retained",
			paths: make([]string, 1200),
		}
		for i := range src.paths {
			src.paths[i] = fmt.Sprintf("file-%04d", i)
		}
		bindNormalTestAction(km, "retained",
			m.PickerAction(func(*view.Editor) *ui.Picker {
				return ui.NewPicker(e, src)
			}), []command.KeyEvent{char('p')},
		)
		bindNormalTestAction(km, "other",
			m.PickerAction(func(*view.Editor) *ui.Picker {
				return ui.NewPicker(e, fixedPickerSource{
					items: fixedPickerItems(1),
				})
			}), []command.KeyEvent{char('o')},
		)
		m = resize(m, 120, 24)
		m2, cmd := m.Update(tea.KeyPressMsg{Code: 'p', Text: "p"})
		m = m2.(ui.Model)
		m2, cmd = m.Update(firstMsg(cmd))
		m = m2.(ui.Model)
		assert.NotPanics(t, func() { m.View() })
		m = sendKey(sendSpecial(m, tea.KeyEscape), 'o')
		m = feedCmds(m, cmd)
		assert.NotContains(t, stripANSI(m.View().Content), "file-")
		m = sendKey(sendSpecial(m, tea.KeyEscape), 'p')
		assert.Equal(t, 1, src.loads)
		assert.Contains(t, stripANSI(m.View().Content), "1200/1200")
		m = sendSpecial(m, tea.KeyEnd)
		assert.Contains(t, stripANSI(m.View().Content), "file-1199")
	})

	for _, count := range []int{0, 37} {
		t.Run(fmt.Sprintf("reopens %d rows", count), func(t *testing.T) {
			e := view.NewEditor(t.TempDir())
			km := command.NewKeymaps()
			m := ui.New(e, km)
			t.Cleanup(m.Close)
			items := fixedPickerItems(112)
			bindNormalTestAction(km, "picker",
				m.PickerAction(func(*view.Editor) *ui.Picker {
					return ui.NewPicker(e, fixedPickerSource{
						items:    items,
						stateKey: "restore",
					})
				}), []command.KeyEvent{char('p')},
			)
			m = resize(m, 120, 24)
			m = sendSpecial(sendKey(m, 'p'), tea.KeyEnd)
			m = sendSpecial(m, tea.KeyEscape)
			items = items[:count]
			m = sendKey(m, 'p')
			assert.NotPanics(t, func() { m.View() })
			assert.Contains(t, stripANSI(m.View().Content), "CONTENT-111")
			assert.Contains(t, stripANSI(m.View().Content), "112/112")
		})
	}
}

func TestShellAction(t *testing.T) {
	t.Run("opens shell prompt", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_ = km.Register("shell", command.Command{
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				fn := func(*view.Editor, string) error { return nil }
				m.ShellAction("$", fn)(e)
				return command.Result{}
			},
			Modes: view.ModeNormal,
			Keys: map[view.Mode]command.KeyBinding{
				view.ModeAny: {{char('!')}},
			},
		})
		m = resize(m, 80, 24)
		m = sendKey(m, '!')
		out := m.View().Content
		assert.Contains(t, out, "$")
	})
}
