package ui_test

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/builtin"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
)

var bgRE = regexp.MustCompile(`48;2;(\d+;\d+;\d+)`)

func TestStatuslineAllElements(t *testing.T) {
	t.Run("renders file-based status elements", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "note.txt")
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
		e := view.NewEditor(root)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		e.SetRegister('a')
		opts := e.Options()
		opts.StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineSeparator},
			{Element: view.StatusLineFileBaseName},
			{Element: view.StatusLineFileAbsolutePath},
			{Element: view.StatusLinePercent},
			{Element: view.StatusLinePrimaryLen},
			{Element: view.StatusLineFileLineEnding},
		}
		opts.StatusLine.Right = []view.StatusLineItem{
			{Element: view.StatusLineFileIndentStyle},
			{Element: view.StatusLineFileType},
		}
		m := resize(ui.New(e, command.NewKeymaps()), 200, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "note.txt")
		assert.Contains(t, out, "reg=a")
	})
}

func TestStatuslineAltBranches(t *testing.T) {
	t.Run("crlf and tabs indent style", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("package p\r\n"), 0o644))
		e := view.NewEditor(root)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		doc.SetLineEnding(core.LineEndingCRLF)
		doc.SetIndentStyle(core.ParseIndentStyle("\t"))
		opts := e.Options()
		opts.StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineFileLineEnding},
			{Element: view.StatusLineFileIndentStyle},
			{Element: view.StatusLineFileType},
		}
		opts.StatusLine.Right = nil
		m := resize(ui.New(e, command.NewKeymaps()), 200, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "crlf")
		assert.Contains(t, out, "tabs")
		assert.Contains(t, out, "go")
	})
}

func TestStatuslineReadOnly(t *testing.T) {
	t.Run("readonly indicator appears", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		doc.SetReadOnly(true)
		opts := e.Options()
		opts.StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineReadOnly},
		}
		opts.StatusLine.Right = nil
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "[readonly]")
	})
}

func TestStatuslineDiagnostics(t *testing.T) {
	t.Run("renders diagnostic counts", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		doc.ReplaceDiagnostics("gopls", []view.Diagnostic{
			{Severity: view.DiagnosticSeverityError},
			{Severity: view.DiagnosticSeverityWarning},
			{Severity: view.DiagnosticSeverityHint},
		})
		e.Options().StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineDiagnostics},
		}
		e.Options().StatusLine.Right = nil
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "E:1")
		assert.Contains(t, out, "W:1")
		assert.Contains(t, out, "H:1")
	})

	t.Run("info and hint counts", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		doc.ReplaceDiagnostics("gopls", []view.Diagnostic{
			{Severity: view.DiagnosticSeverityInfo},
			{Severity: view.DiagnosticSeverityHint},
		})
		e.Options().StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineDiagnostics},
		}
		e.Options().StatusLine.Right = nil
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "I:1")
		assert.Contains(t, out, "H:1")
	})
}

func TestThemeStyleBranches(t *testing.T) {
	t.Run("ao theme covers extended style keys", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		e := view.NewEditor(t.TempDir())
		e.Options().Theme = "ao"
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := m.View().Content

		assert.NotEmpty(t, out)
	})
}

func TestModeColorRender(t *testing.T) {
	t.Run("applies mode color", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		e := view.NewEditor(t.TempDir())
		e.Options().Theme = "mocha"
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := m.View().Content

		assert.Contains(t, out, "\x1b[48;2;245;224;220m NOR ")
	})
}

func TestSpinnerColorRender(t *testing.T) {
	t.Run("applies spinner color", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		e := view.NewEditor(t.TempDir())
		e.Options().Theme = "mocha"
		e.SetLanguageServerController(&completionController{busy: true})
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := m.View().Content

		assert.Contains(t, out, "\x1b[38;2;137;180;250m")
	})

	t.Run("busy transition redraws spinner", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		ctl := &completionController{}
		e.SetLanguageServerController(ctl)
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		_ = m.View()

		batch, ok := m.Init()().(tea.BatchMsg)
		assert.True(t, ok)
		var redraw tea.Cmd
		for _, cmd := range batch {
			if msg, ok := runWithTimeout(cmd, 20*time.Millisecond); ok {
				next, cmd := m.Update(msg)
				m = next.(ui.Model)
				redraw = cmd
			}
		}
		if !assert.NotNil(t, redraw) {
			return
		}

		ctl.busy = true
		e.Tree().Redraw()
		msg, ok := runWithTimeout(redraw, time.Second)
		if !assert.True(t, ok) {
			return
		}
		next, _ := m.Update(msg)
		m = next.(ui.Model)

		assert.Contains(t, stripANSI(m.View().Content), "⠋")
	})
}

func TestStatuslineSpacing(t *testing.T) {
	t.Run("custom spinner location", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.SetLanguageServerController(&completionController{busy: true})
		e.Options().StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineFileName},
			{Element: view.StatusLineSpinner},
		}
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "[scratch]  ⠋ ")
		assert.NotContains(t, out, "[scratch]   ⠋")
	})
}

func TestStatuslineTotalLines(t *testing.T) {
	t.Run("total-line-numbers appears in status", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.Options().StatusLine.Right = []view.StatusLineItem{
			{Element: view.StatusLineTotalLines},
		}
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, " 1 ")
	})
}

func TestStatuslineElementRegistry(t *testing.T) {
	cases := []struct {
		element view.StatusLineElement
		setup   func(t *testing.T) *view.Editor
		left    []view.StatusLineItem
		want    string
	}{
		{element: view.StatusLineMode, want: " NOR "},
		{element: view.StatusLineSeparator, want: "│"},
		{
			element: view.StatusLineFileBaseName,
			setup:   fileEditor,
			want:    "note.txt",
		},
		{
			element: view.StatusLineFileName,
			setup:   fileEditor,
			want:    "note.txt",
		},
		{
			element: view.StatusLineFileAbsolutePath,
			setup:   fileEditor,
			want:    "note.txt",
		},
		{
			element: view.StatusLineReadOnly,
			setup: func(t *testing.T) *view.Editor {
				e := view.NewEditor(t.TempDir())
				doc := e.FocusedDocument()
				assert.NotNil(t, doc)
				doc.SetReadOnly(true)
				return e
			},
			want: "[readonly]",
		},
		{
			element: view.StatusLineModified,
			setup: func(t *testing.T) *view.Editor {
				return editorWithText(t, "changed")
			},
			want: "[modified]",
		},
		{element: view.StatusLineFileEncoding, want: " utf-8 "},
		{element: view.StatusLineFileLineEnding, want: " lf "},
		{
			element: view.StatusLineFileIndentStyle,
			setup: func(t *testing.T) *view.Editor {
				e := view.NewEditor(t.TempDir())
				doc := e.FocusedDocument()
				assert.NotNil(t, doc)
				doc.SetIndentStyle(core.ParseIndentStyle("\t"))
				return e
			},
			want: " tabs ",
		},
		{element: view.StatusLineFileType, want: " text "},
		{
			element: view.StatusLineDiagnostics,
			setup: func(t *testing.T) *view.Editor {
				e := view.NewEditor(t.TempDir())
				doc := e.FocusedDocument()
				assert.NotNil(t, doc)
				doc.ReplaceDiagnostics("gopls", []view.Diagnostic{
					{Severity: view.DiagnosticSeverityError},
				})
				return e
			},
			want: "E:1",
		},
		{element: view.StatusLineSelections, want: " 1 sel "},
		{
			element: view.StatusLinePrimaryLen,
			setup: func(t *testing.T) *view.Editor {
				e := editorWithText(t, "abcd")
				testutil.SetSelection(t, e,
					[]core.Range{{Anchor: 0, Head: 2}}, 0,
				)
				return e
			},
			want: " 2 ",
		},
		{element: view.StatusLinePosition, want: " 1:1 "},
		{element: view.StatusLinePercent, want: "%"},
		{element: view.StatusLineTotalLines, want: " 1 "},
		{
			element: view.StatusLineSpacer,
			left: []view.StatusLineItem{
				{Element: view.StatusLineFileType},
				{Element: view.StatusLineSpacer},
				{Element: view.StatusLineFileType},
			},
			want: "text   text",
		},
		{
			element: view.StatusLineVersionControl,
			setup: func(t *testing.T) *view.Editor {
				testutil.RequireGit(t)
				e, s := repoEditor(t, "one\n", "one\nCHANGED\n")
				t.Cleanup(s.Close)
				return e
			},
			want: " main ",
		},
		{
			element: view.StatusLineSpinner,
			setup: func(t *testing.T) *view.Editor {
				e := view.NewEditor(t.TempDir())
				e.SetLanguageServerController(&completionController{busy: true})
				return e
			},
			want: "⠋",
		},
	}

	for _, tc := range cases {
		t.Run(string(tc.element), func(t *testing.T) {
			e := view.NewEditor(t.TempDir())
			if tc.setup != nil {
				e = tc.setup(t)
			}
			left := tc.left
			if left == nil {
				left = []view.StatusLineItem{{Element: tc.element}}
			}
			e.Options().StatusLine.Left = left
			e.Options().StatusLine.Right = []view.StatusLineItem{
				{Element: view.StatusLineSpacer},
			}
			m := resize(ui.New(e, command.NewKeymaps()), 200, 24)

			out := stripANSI(m.View().Content)

			assert.Contains(t, out, tc.want)
		})
	}
}

func TestStatuslineEncoding(t *testing.T) {
	t.Run("utf-8 without bom", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "plain.txt")
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
		e := view.NewEditor(root)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		e.Options().StatusLine.Right = []view.StatusLineItem{
			{Element: view.StatusLineFileEncoding},
		}
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, " utf-8 ")
	})

	t.Run("utf-8 with bom", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "bom.txt")
		data := append([]byte{0xef, 0xbb, 0xbf}, []byte("hello\n")...)
		assert.NoError(t, os.WriteFile(path, data, 0o644))
		e := view.NewEditor(root)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		e.Options().StatusLine.Right = []view.StatusLineItem{
			{Element: view.StatusLineFileEncoding},
		}
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, " utf-8-bom ")
	})
}

func TestStatuslineEdgeElements(t *testing.T) {
	t.Run("narrow width drops rightmost first", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "very-long-file-name.txt")
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
		e := view.NewEditor(root)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		opts := e.Options()
		opts.StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineMode, Pinned: true},
			{Element: view.StatusLineSelections},
			{Element: view.StatusLineFileAbsolutePath},
		}
		opts.StatusLine.Right = []view.StatusLineItem{
			{Element: view.StatusLineSpacer},
		}
		m := resize(ui.New(e, command.NewKeymaps()), 14, 8)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, " NOR ")
		assert.Contains(t, out, "1 sel")
		assert.NotContains(t, out, "very-long-file-name")
	})

	t.Run("right section drops from its left", func(t *testing.T) {
		e := editorWithText(t, "hello")
		opts := e.Options()
		opts.StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineSpacer},
		}
		opts.StatusLine.Right = []view.StatusLineItem{
			{Element: view.StatusLineFileType},
			{Element: view.StatusLinePosition},
		}
		m := resize(ui.New(e, command.NewKeymaps()), 7, 8)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, " 1:1 ")
		assert.NotContains(t, out, "text")
	})

	t.Run("pinned element survives narrow width", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "very-long-file-name.txt")
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
		e := view.NewEditor(root)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		opts := e.Options()
		opts.StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineMode},
			{Element: view.StatusLineFileAbsolutePath},
		}
		opts.StatusLine.Right = []view.StatusLineItem{
			{Element: view.StatusLinePosition, Pinned: true},
		}
		m := resize(ui.New(e, command.NewKeymaps()), 12, 8)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, " 1:1 ")
		assert.NotContains(t, out, "very-long-file-name")
	})

	t.Run("modified scratch appears", func(t *testing.T) {
		e := editorWithText(t, "changed")
		e.Options().StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineModified},
		}
		e.Options().StatusLine.Right = nil
		m := resize(ui.New(e, command.NewKeymaps()), 80, 8)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "[modified]")
	})

	t.Run("plural selections include primary", func(t *testing.T) {
		e := editorWithText(t, "abcd")
		testutil.SetSelection(t, e,
			[]core.Range{core.PointRange(0), core.PointRange(2)},
			1,
		)
		e.Options().StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineSelections},
			{Element: view.StatusLinePrimaryLen},
		}
		e.Options().StatusLine.Right = nil
		m := resize(ui.New(e, command.NewKeymaps()), 80, 8)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, " 2/2 sels ")
		assert.Contains(t, out, " 0 ")
	})

	t.Run("spaces indent and default file type", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		doc.SetIndentStyle(core.Spaces(2))
		doc.SetLang("")
		e.Options().StatusLine.Left = []view.StatusLineItem{
			{Element: view.StatusLineFileIndentStyle},
			{Element: view.StatusLineFileType},
		}
		e.Options().StatusLine.Right = nil
		m := resize(ui.New(e, command.NewKeymaps()), 80, 8)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, " spaces:2 ")
		assert.Contains(t, out, " text ")
	})
}

func TestCommandlineThemeRender(t *testing.T) {
	t.Run("applies commandline styles", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		e := view.NewEditor(root)
		e.Options().Theme = "mocha"
		km := command.NewKeymaps()
		assert.NoError(t, km.Register("message", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				return command.Result{Message: "error: harmless"}
			},
			Modes:   view.ModeNormal,
			Aliases: []string{"message"},
		}))
		assert.NoError(t, km.Register("failure", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				return command.Result{Error: assert.AnError}
			},
			Modes:   view.ModeNormal,
			Aliases: []string{"failure"},
		}))
		assert.NoError(t, km.Register("empty", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				return command.Result{}
			},
			Modes:   view.ModeNormal,
			Aliases: []string{"empty"},
			Keys: map[view.Mode]command.KeyBinding{
				view.ModeAny: {{char('x')}},
			},
		}))
		m := resize(ui.New(e, km), 80, 24)

		prompt := sendKey(m, ':').View().Content
		errOut := m.ExecTypable("failure").View().Content
		m = m.ExecTypable("message")
		msgOut := m.View().Content
		m = m.ExecTypable("empty")
		emptyOut := m.View().Content
		m = sendKey(m, 'x')
		clearedOut := m.View().Content

		assert.Contains(t, prompt, "\x1b[38;2;205;214;244m")
		assert.Contains(t, errOut, "\x1b[38;2;243;139;168m")
		assert.Contains(t, msgOut, "\x1b[38;2;205;214;244m")
		assert.Contains(t, stripANSI(emptyOut), "error: harmless")
		assert.NotContains(t, stripANSI(clearedOut), "error: harmless")
	})
}

func fileEditor(t *testing.T) *view.Editor {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "note.txt")
	assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
	e := view.NewEditor(root)
	_, err := e.OpenFile(path)
	assert.NoError(t, err)
	return e
}

func TestCmdlineHighlight(t *testing.T) {
	t.Run("an interaction uses the popup", func(t *testing.T) {
		m := builtinModel(t)
		idle := lastLine(m.View().Content)

		m = sendKey(m, '"')

		assert.Equal(t,
			backgrounds(idle), backgrounds(lastLine(m.View().Content)))
		assert.Equal(t, `"`, popupHead(m))

		m = sendKey(m, 'a')
		assert.Empty(t, popupHead(m))
	})

	t.Run("recording shares prompt background", func(t *testing.T) {
		idle := lastLine(builtinModel(t).View().Content)
		rows := promptRows(sendKey(builtinModel(t), ':'))
		assert.NotEmpty(t, rows)
		prompt := rows[0].raw

		m := sendKey(sendKey(builtinModel(t), 'Q'), 'q')
		recording := lastLine(m.View().Content)

		assert.NotEqual(t, backgrounds(idle), backgrounds(recording))
		assert.Contains(t, backgrounds(recording), backgrounds(prompt)[0])
		assert.Contains(t, stripANSI(recording), "REC q")
	})

	t.Run("pending keys leave the cmdline idle", func(t *testing.T) {
		idle := lastLine(builtinModel(t).View().Content)
		pending := lastLine(sendKey(builtinModel(t), ' ').View().Content)

		assert.Equal(t, backgrounds(idle), backgrounds(pending))
		assert.NotContains(t, stripANSI(pending), "spc")
	})

	t.Run("a count leaves the cmdline idle", func(t *testing.T) {
		idle := lastLine(builtinModel(t).View().Content)
		counted := lastLine(sendKey(builtinModel(t), '3').View().Content)

		assert.Equal(t, backgrounds(idle), backgrounds(counted))
		assert.NotContains(t, stripANSI(counted), "3")
	})
}

func TestStatuslineRegisterFeedback(t *testing.T) {
	t.Run("selecting a register shows it at once", func(t *testing.T) {
		m := builtinModel(t)
		assert.NotContains(t, stripANSI(m.View().Content), "reg=")

		m = sendKey(sendKey(m, '"'), '5')

		assert.Contains(t, stripANSI(lastLine(m.View().Content)), "reg=5")
	})

	t.Run("one register badge, not one per pane", func(t *testing.T) {
		m := builtinModel(t)
		m = sendKey(sendKey(m, '"'), '5')
		m = sendKey(sendKey(m, ' '), 'w')
		m = sendKey(m, 'v') // vertical split

		out := stripANSI(m.View().Content)

		assert.Equal(t, 1, strings.Count(out, "reg=5"))
		assert.Contains(t, stripANSI(lastLine(out)), "reg=5")
	})
}

func TestPendingKeyPopup(t *testing.T) {
	t.Run("continuation leaves keep their labels", func(t *testing.T) {
		m := sendKey(builtinModel(t), 'r')
		lines := strings.Split(stripANSI(m.View().Content), "\n")
		top := lineIndexWith(lines, "╭")
		assert.GreaterOrEqual(t, top, 0)
		assert.Contains(t, lines[top], "Replace with new char")

		m = sendKey(builtinModel(t), 'i')
		m = sendModified(m, 'r', tea.ModCtrl)
		lines = strings.Split(stripANSI(m.View().Content), "\n")
		top = lineIndexWith(lines, "╭")
		assert.GreaterOrEqual(t, top, 0)
		assert.Contains(t, lines[top], "Insert register")
	})

	t.Run("the title rides the top border", func(t *testing.T) {
		m := sendKey(builtinModel(t), ' ')
		lines := strings.Split(stripANSI(m.View().Content), "\n")
		top := lineIndexWith(lines, "\u256d")
		assert.GreaterOrEqual(t, top, 0)
		assert.Contains(t, lines[top], "Leader")
		assert.Equal(t, "spc", popupHead(m))
		assert.NotContains(t, lines[top+1], "Leader")
	})

	t.Run("an interaction wears its command's label", func(t *testing.T) {
		m := sendKey(builtinModel(t), '"')
		lines := strings.Split(stripANSI(m.View().Content), "\n")
		top := lineIndexWith(lines, "╭")
		assert.GreaterOrEqual(t, top, 0)
		assert.Contains(t, lines[top], "Select register")
		assert.Equal(t, `"`, popupHead(m))
	})

	t.Run("interaction keeps count and choices", func(t *testing.T) {
		m := sendKey(sendKey(sendKey(builtinModel(t), '5'), 'm'), 'r')
		lines := strings.Split(stripANSI(m.View().Content), "\n")
		top := lineIndexWith(lines, "╭")
		assert.GreaterOrEqual(t, top, 0)
		assert.Contains(t, lines[top], "Surround replace")
		assert.Equal(t, "m r → 5", popupHead(m))
		assert.Contains(t, lines[top+3], "parentheses")
	})

	t.Run("register prompt previews values", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)
		e.Registers().Set('a', "hello\nthere")

		m = sendKey(m, '"')

		lines := strings.Split(stripANSI(m.View().Content), "\n")
		top := lineIndexWith(lines, "╭")
		assert.GreaterOrEqual(t, top, 0)
		assert.Contains(t, lines[top+3], "a")
		assert.Contains(t, lines[top+3], "hello there")
	})

	t.Run("static interaction help comes from trie", func(t *testing.T) {
		m := sendKey(sendKey(sendKey(builtinModel(t), ' '), 'w'), 'r')
		out := stripANSI(m.View().Content)

		assert.Equal(t, "spc w r", popupHead(m))
		assert.Contains(t, out, "Resize split")
		assert.Contains(t, out, "resize, esc/enter exits")

		m = sendKey(m, 'h')
		assert.Equal(t, "spc w r", popupHead(m))

		m = sendSpecial(m, tea.KeyEscape)
		assert.Empty(t, popupHead(m))
	})

	t.Run("deeper sequences space the keys", func(t *testing.T) {
		m := sendKey(sendKey(builtinModel(t), ' '), 'w')
		assert.Equal(t, "spc w", popupHead(m))
	})

	t.Run("uncounted menus reject counts", func(t *testing.T) {
		m := sendKey(sendKey(builtinModel(t), ' '), 'w')
		m = sendKey(m, '9')

		assert.Equal(t, "spc w", popupHead(m))
		assert.Contains(t, stripANSI(m.View().Content), "Close window")
	})

	t.Run("wheel belongs to the menu", func(t *testing.T) {
		e := editorWithText(t, strings.Repeat("line\n", 40))
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 100, 30)
		m = sendKey(sendKey(m, ' '), 'w')

		before := stripANSI(m.View().Content)
		v := e.FocusedView()
		assert.NotNil(t, v)
		off := v.Offset()
		m = mouse(m, tea.MouseWheelMsg{Button: tea.MouseWheelDown})
		after := stripANSI(m.View().Content)

		assert.Equal(t, before, after)
		assert.Equal(t, off, v.Offset())
	})

	t.Run("click follows picker dismissal", func(t *testing.T) {
		m := sendKey(sendKey(builtinModel(t), ' '), 'w')
		lines := strings.Split(stripANSI(m.View().Content), "\n")
		top := lineIndexWith(lines, "╭")

		m = mouse(m, tea.MouseClickMsg{
			X:      50,
			Y:      top + 3,
			Button: tea.MouseLeft,
		})
		assert.Equal(t, "spc w", popupHead(m))

		m = mouse(m, tea.MouseClickMsg{Button: tea.MouseLeft})
		assert.Empty(t, popupHead(m))
	})

	t.Run("escape closes the menu", func(t *testing.T) {
		m := sendKey(sendKey(builtinModel(t), ' '), 'w')
		m = sendSpecial(m, tea.KeyEscape)

		assert.Empty(t, popupHead(m))
	})

	t.Run("errant prompt keys keep it open", func(t *testing.T) {
		m := sendKey(builtinModel(t), 'r')
		m = sendSpecial(m, tea.KeyUp)

		assert.Equal(t, "r", popupHead(m))
	})

	t.Run("hints sit under a rule", func(t *testing.T) {
		m := sendKey(builtinModel(t), ' ')
		lines := strings.Split(stripANSI(m.View().Content), "\n")
		top := lineIndexWith(lines, "\u256d")
		assert.GreaterOrEqual(t, top, 0)
		assert.Contains(t, lines[top+2], "\u251c")
		assert.Contains(t, lines[top+3], "Window")
	})

	t.Run("a hintless node is breadcrumb only", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(km, "dd", func(*view.Editor) {},
			[]command.KeyEvent{char('d'), char('d')})
		m = resize(m, 100, 30)

		m = sendKey(m, 'd')
		lines := strings.Split(stripANSI(m.View().Content), "\n")
		top := lineIndexWith(lines, "\u256d")
		assert.Equal(t, "d", popupHead(m))
		assert.Contains(t, lines[top+2], "\u2570")
	})

	t.Run("backspace pops the last key", func(t *testing.T) {
		m := sendKey(sendKey(builtinModel(t), ' '), 'w')
		assert.Equal(t, "spc w", popupHead(m))

		m = sendSpecial(m, tea.KeyBackspace)

		assert.Equal(t, "spc", popupHead(m))
		lines := strings.Split(stripANSI(m.View().Content), "\n")
		top := lineIndexWith(lines, "╭")
		assert.Contains(t, lines[top], "Leader")
		assert.Contains(t, lines[top+3], "Window")
	})

	t.Run("continuation may keep backspace", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestKeyAction(km, "keep",
			func(*view.Editor) command.Continuation {
				var cont command.Continuation
				cont = func(
					*view.Editor, command.KeyEvent,
				) (command.Continuation, command.Transition) {
					return cont, command.ContinuationStay
				}
				return cont
			},
			[]command.KeyEvent{char('x')},
		)
		m = resize(m, 100, 30)

		m = sendKey(m, 'x')
		m = sendSpecial(m, tea.KeyBackspace)

		assert.Equal(t, "x", popupHead(m))
	})

	t.Run("continuation pops to its parent", func(t *testing.T) {
		m := sendKey(sendKey(sendKey(builtinModel(t), ' '), 'w'), 'r')
		assert.Equal(t, "spc w r", popupHead(m))

		m = sendSpecial(m, tea.KeyBackspace)

		assert.Equal(t, "spc w", popupHead(m))
	})

	t.Run("nested continuation backtracks", func(t *testing.T) {
		m := sendKey(sendKey(builtinModel(t), 'm'), 'r')
		m = sendKey(m, '(')
		assert.Equal(t, "m r (", popupHead(m))
		assert.Contains(t, stripANSI(m.View().Content), "Surround replace")
		assert.NotContains(t, stripANSI(m.View().Content), "parentheses")

		m = sendSpecial(m, tea.KeyBackspace)
		assert.Equal(t, "m r", popupHead(m))
		assert.Contains(t, stripANSI(m.View().Content), "parentheses")

		m = sendSpecial(m, tea.KeyBackspace)
		assert.Equal(t, "m", popupHead(m))
	})

	t.Run("backspace pops count digits before keys", func(t *testing.T) {
		m := sendKey(sendKey(sendKey(builtinModel(t), 'g'), '6'), '2')
		assert.Equal(t, "g → 62", popupHead(m))

		m = sendSpecial(m, tea.KeyBackspace)
		assert.Equal(t, "g → 6", popupHead(m))

		m = sendSpecial(m, tea.KeyBackspace)
		assert.Equal(t, "g", popupHead(m))
	})

	t.Run("backspace follows input order", func(t *testing.T) {
		m := sendKey(sendKey(builtinModel(t), '2'), 'g')
		assert.Equal(t, "g → 2", popupHead(m))

		m = sendSpecial(m, tea.KeyBackspace)
		assert.Equal(t, "2", popupHead(m))

		m = sendSpecial(m, tea.KeyBackspace)
		assert.Empty(t, popupHead(m))
	})

	t.Run("a bare count opens the popup", func(t *testing.T) {
		m := sendKey(sendKey(builtinModel(t), '4'), '2')
		assert.Equal(t, "42", popupHead(m))
		lines := strings.Split(stripANSI(m.View().Content), "\n")
		top := lineIndexWith(lines, "╭")
		assert.Contains(t, lines[top], i18n.Text(i18n.StatusCounted))

		m = sendSpecial(m, tea.KeyBackspace)
		assert.Equal(t, "4", popupHead(m))

		m = sendSpecial(m, tea.KeyBackspace)
		assert.Empty(t, popupHead(m))
	})

	t.Run("popping the last key closes the popup", func(t *testing.T) {
		m := sendKey(builtinModel(t), ' ')
		assert.Equal(t, "spc", popupHead(m))

		m = sendSpecial(m, tea.KeyBackspace)

		lines := strings.Split(stripANSI(m.View().Content), "\n")
		assert.Equal(t, -1, lineIndexWith(lines, "╭"))
		assert.Equal(t, -1, lineIndexWith(lines, "Leader"))
	})

	t.Run("backspace stays bound when idle", func(t *testing.T) {
		e := editorWithText(t, "abc")
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(km, "del", func(e *view.Editor) {
			e.SetStatusMsg("deleted")
		}, []command.KeyEvent{{Code: command.KeyCode{
			Special: command.Backspace,
		}}})
		m = resize(m, 100, 30)

		m = sendSpecial(m, tea.KeyBackspace)

		assert.Contains(t, stripANSI(lastLine(m.View().Content)), "deleted")
	})

	t.Run("a long menu clears the status rows", func(t *testing.T) {
		m := sendKey(builtinModel(t), ' ')
		lines := strings.Split(stripANSI(m.View().Content), "\n")
		top := lineIndexWith(lines, "\u256d")
		bottom := lineIndexWith(lines, "\u2570")
		assert.GreaterOrEqual(t, top, 0)
		assert.Less(t, bottom, len(lines)-2)
		assert.Contains(t, lines[len(lines)-2], "1 sel")

		box := []rune(lines[top])
		left := slices.Index(box, '\u256d')
		right := len(box) - slices.Index(box, '\u256e') - 1
		assert.InDelta(t, left, right, 1)
	})

	t.Run("an interaction spares a stale message", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 100, 30)
		e.SetStatusMsg("older news")
		m = resize(m, 100, 30)
		assert.Contains(t, stripANSI(lastLine(m.View().Content)), "older news")

		m = sendKey(m, ' ')

		assert.Contains(t, stripANSI(lastLine(m.View().Content)), "older news")
		assert.Equal(t, "spc", popupHead(m))
	})
}

func lineIndexWith(lines []string, want string) int {
	for i, line := range lines {
		if strings.Contains(line, want) {
			return i
		}
	}
	return -1
}

func popupHead(m ui.Model) string {
	lines := strings.Split(stripANSI(m.View().Content), "\n")
	top := lineIndexWith(lines, "╭")
	if top < 0 || top+1 >= len(lines) {
		return ""
	}
	return strings.TrimSpace(strings.Trim(lines[top+1], "│ "))
}

func builtinModel(t *testing.T) ui.Model {
	t.Helper()
	e := view.NewEditor(t.TempDir())
	km := command.NewKeymaps()
	m := ui.New(e, km)
	_, err := builtin.Register(m, km)
	assert.NoError(t, err)
	return resize(m, 100, 30)
}

func lastLine(content string) string {
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	return lines[len(lines)-1]
}

func backgrounds(line string) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range bgRE.FindAllStringSubmatch(line, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	return out
}
