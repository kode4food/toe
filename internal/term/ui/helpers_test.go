package ui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/term/builtin/files"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

type feedPickerSource struct {
	ui.PickerBase
	paths []string
}

func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;:?]*[ -/]*[@-~]`)
	return strings.TrimRight(re.ReplaceAllString(s, ""), "\n")
}

func assertPromptCountRightPadding(t *testing.T, out string) {
	t.Helper()
	re := regexp.MustCompile(`[0-9]+/[0-9]+ `)
	assert.True(t, re.MatchString(out))
}

func previewPaneLine(t *testing.T, content, want string) string {
	t.Helper()
	for raw := range strings.SplitSeq(content, "\n") {
		stripped := stripANSI(raw)
		if !strings.Contains(stripped, "│") {
			continue
		}
		for col := range strings.SplitSeq(stripped, "│") {
			if strings.TrimSpace(col) == want {
				return raw
			}
		}
	}
	t.Fatalf("no preview pane row found containing %q", want)
	return ""
}

func rawLineContaining(t *testing.T, content, want string) string {
	t.Helper()
	for raw := range strings.SplitSeq(content, "\n") {
		if strings.Contains(stripANSI(raw), want) {
			return raw
		}
	}
	t.Fatalf("no row found containing %q", want)
	return ""
}

func assertRenderedWidth(t *testing.T, out string, w int) {
	t.Helper()
	for line := range strings.SplitSeq(out, "\n") {
		assert.LessOrEqual(t, ansi.StringWidth(line), w)
	}
}

func bufferPicker(e *view.Editor) *ui.Picker {
	return files.NewBufferPicker(e, files.BufferPickerOptions{
		StartPosition: files.PickerStartTop,
	})
}

func bufferDirExplorer(e *view.Editor) *ui.Picker {
	return files.NewFocusedPaneDirExplorer(
		e, files.DefaultFileExplorerOptions(),
	)
}

func writeLanguageConfig(t *testing.T, root, lang string, enabled bool) {
	t.Helper()
	dir := filepath.Join(root, loader.DirName)
	err := os.MkdirAll(dir, 0o755)
	assert.NoError(t, err)
	text := "[[language]]\n" +
		"name = \"" + lang + "\"\n" +
		fmt.Sprintf("soft-wrap.enable = %t\n", enabled)
	err = os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	)
	assert.NoError(t, err)
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writeConfig(t *testing.T, root, text string) {
	t.Helper()
	dir := filepath.Join(root, loader.DirName)
	err := os.MkdirAll(dir, 0o755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "config.toml"), []byte(text), 0o644)
	assert.NoError(t, err)
}

func writeConfigIgnore(t *testing.T, root, text string) {
	t.Helper()
	dir := filepath.Join(root, loader.DirName)
	err := os.MkdirAll(dir, 0o755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "ignore"), []byte(text), 0o644)
	assert.NoError(t, err)
	t.Setenv("XDG_CONFIG_HOME", root)
}

func sendKey(m ui.Model, ch rune) ui.Model {
	m2, _ := m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	return m2.(ui.Model)
}

func sendKeyAndFeed(m ui.Model, ch rune) ui.Model {
	return updateAndFeed(m, tea.KeyPressMsg{Code: ch, Text: string(ch)})
}

func updateAndFeed(m ui.Model, msg tea.Msg) ui.Model {
	m2, cmd := m.Update(msg)
	m = m2.(ui.Model)
	return feedCmds(m, cmd)
}

func feedCmds(m ui.Model, cmd tea.Cmd) ui.Model {
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			break
		}
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, next := range batch {
				m = feedCmds(m, next)
			}
			break
		}
		m2, next := m.Update(msg)
		m = m2.(ui.Model)
		cmd = next
	}
	return m
}

func sendSpecial(m ui.Model, k rune) ui.Model {
	m2, _ := m.Update(tea.KeyPressMsg{Code: k})
	return m2.(ui.Model)
}

func sendSpecialAndFeed(m ui.Model, k rune) ui.Model {
	return updateAndFeed(m, tea.KeyPressMsg{Code: k})
}

func sendSpecialText(m ui.Model, k rune, text string) ui.Model {
	m2, _ := m.Update(tea.KeyPressMsg{Code: k, Text: text})
	return m2.(ui.Model)
}

func sendModified(m ui.Model, ch rune, mod tea.KeyMod) ui.Model {
	m2, _ := m.Update(tea.KeyPressMsg{Code: ch, Mod: mod})
	return m2.(ui.Model)
}

func sendModifiedAndFeed(m ui.Model, ch rune, mod tea.KeyMod) ui.Model {
	return updateAndFeed(m, tea.KeyPressMsg{Code: ch, Mod: mod})
}

func resize(m ui.Model, w, h int) ui.Model {
	m2, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return m2.(ui.Model)
}

func mouse(m ui.Model, msg tea.Msg) ui.Model {
	m2, _ := m.Update(msg)
	return m2.(ui.Model)
}

type bindTestActionArgs struct {
	km   *command.Keymaps
	mode view.Mode
	name string
	fn   command.KeyAction
	seqs [][]command.KeyEvent
}

func bindTestAction(args bindTestActionArgs) {
	_ = args.km.Register(args.name, command.Command{
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			return command.Result{Continuation: args.fn(e)}
		},
		Modes: args.mode,
		Keys:  map[view.Mode]command.KeyBinding{view.ModeAny: args.seqs},
	})
}

func bindNormalTestAction(
	km *command.Keymaps, name string, fn command.Action,
	seqs ...[]command.KeyEvent,
) {
	bindTestAction(bindTestActionArgs{
		km:   km,
		mode: view.ModeNormal,
		name: name,
		fn: func(e *view.Editor) command.Continuation {
			fn(e)
			return nil
		},
		seqs: seqs,
	})
}

// bindNormalTestKeyAction binds a continuation-returning action, for the
// multi-key cases bindNormalTestAction's plain Action cannot express
func bindNormalTestKeyAction(
	km *command.Keymaps, name string, fn command.KeyAction,
	seqs ...[]command.KeyEvent,
) {
	bindTestAction(bindTestActionArgs{
		km: km, mode: view.ModeNormal, name: name, fn: fn, seqs: seqs,
	})
}

func char(ch rune) command.KeyEvent {
	return command.KeyEvent{Code: command.KeyCode{Char: ch}}
}

func special(s command.Special) command.KeyEvent {
	return command.KeyEvent{Code: command.KeyCode{Special: s}}
}

func (s feedPickerSource) Load(
	*view.Editor,
) ([]ui.PickerItem, <-chan ui.PickerItem, ui.StopFunc) {
	ch := make(chan ui.PickerItem, len(s.paths))
	for _, p := range s.paths {
		ch <- ui.PickerItem{
			Display:  p,
			SortKey:  p,
			Location: ui.PickerLocation{Target: ui.PickerTarget{Path: p}},
		}
	}
	close(ch)
	return nil, ch, func() {}
}

func (feedPickerSource) Accept(
	*view.Editor, *ui.PickerItem, ui.PickerAcceptAction,
) {
}

func (feedPickerSource) SkipPreview() {}

func feedPickerModel(t testing.TB, paths []string) ui.Model {
	// the picker is bound to 'p' but not yet opened, so callers can time
	// the open and its feed themselves
	t.Helper()
	src := feedPickerSource{
		PickerBase: ui.NewPickerBase("feed", []string{"path"}, 0, nil),
		paths:      paths,
	}
	e := view.NewEditor(t.TempDir())
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "feed_picker",
		m.PickerAction(func(*view.Editor) *ui.Picker {
			return ui.NewPicker(e, src)
		}),
		[]command.KeyEvent{char('p')},
	)
	return resize(m, 100, 30)
}
