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
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

type bindTestActionArgs struct {
	km   *command.Keymaps
	mode string
	name string
	fn   command.KeyAction
	seqs [][]command.KeyEvent
}

func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
	return strings.TrimRight(re.ReplaceAllString(s, ""), "\n")
}

func assertPromptCountRightPadding(t *testing.T, out string) {
	t.Helper()
	re := regexp.MustCompile(`[0-9]+/[0-9]+ `)
	assert.True(t, re.MatchString(out))
}

func assertRenderedWidth(t *testing.T, out string, w int) {
	t.Helper()
	for line := range strings.SplitSeq(out, "\n") {
		assert.LessOrEqual(t, ansi.StringWidth(line), w)
	}
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

func openPickerAndFeed(m ui.Model, ch rune) ui.Model {
	m2, cmd := m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	m = m2.(ui.Model)
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			break
		}
		m2, cmd = m.Update(msg)
		m = m2.(ui.Model)
	}
	return m
}

func sendSpecial(m ui.Model, k rune) ui.Model {
	m2, _ := m.Update(tea.KeyPressMsg{Code: k})
	return m2.(ui.Model)
}

func sendSpecialText(m ui.Model, k rune, text string) ui.Model {
	m2, _ := m.Update(tea.KeyPressMsg{Code: k, Text: text})
	return m2.(ui.Model)
}

func resize(m ui.Model, w, h int) ui.Model {
	m2, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return m2.(ui.Model)
}

func bindTestAction(args bindTestActionArgs) {
	_ = args.km.Register(args.name, command.Command{
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			return command.Result{Continuation: args.fn(e)}
		},
		Modes: []string{args.mode},
		Keys:  map[string][]command.KeyBinding{"*": {args.seqs}},
	})
}

func bindNormalTestAction(
	km *command.Keymaps, name string, fn command.KeyAction,
	seqs ...[]command.KeyEvent,
) {
	bindTestAction(bindTestActionArgs{
		km: km, mode: "NOR", name: name, fn: fn, seqs: seqs,
	})
}
