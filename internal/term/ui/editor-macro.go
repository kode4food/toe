package ui

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/tui"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

type macroSlot struct {
	recording   bool
	reg         rune
	keys        []command.KeyEvent
	replayReg   rune
	replayCount int
	hasReplay   bool
}

// MacroRecordAction starts or stops macro recording. When not recording,
// prompts for a register key and begins recording. When already recording,
// stops and saves the macro to the chosen register
func (e *EditorComponent) MacroRecordAction(
	ed *view.Editor,
) command.Continuation {
	ms := e.macroSlot
	if ms.recording {
		if len(ms.keys) > 0 {
			ms.keys = ms.keys[:len(ms.keys)-1]
		}
		ed.Registers().Set(ms.reg, macroText(ms.keys))
		ms.recording = false
		ms.keys = nil
		ms.reg = 0
		return nil
	}
	ed.SetHint("Q ...")
	return func(ed *view.Editor, k command.KeyEvent) command.Continuation {
		if k.Code.Char == 0 || k.Mods != command.ModNone {
			ed.SetHint("")
			return nil
		}
		ms.recording = true
		ms.reg = k.Code.Char
		ms.keys = nil
		return nil
	}
}

// MacroReplayAction prompts for a register key and replays the macro stored
// there count times
func (e *EditorComponent) MacroReplayAction(
	ed *view.Editor,
) command.Continuation {
	ms := e.macroSlot
	if ms.recording {
		return nil
	}
	ed.SetHint("q ...")
	return func(ed *view.Editor, k command.KeyEvent) command.Continuation {
		if k.Code.Char == 0 || k.Mods != command.ModNone {
			ed.SetHint("")
			return nil
		}
		n := ed.Count()
		if n == 0 {
			n = 1
		}
		ms.replayReg = k.Code.Char
		ms.replayCount = n
		ms.hasReplay = true
		return nil
	}
}

func (e *EditorComponent) handleReplay(cx *Context) {
	ms := e.macroSlot
	if !ms.hasReplay {
		return
	}
	ms.hasReplay = false
	e.replayMacro(cx, macroKeys(cx.Editor, ms.replayReg), ms.replayCount)
	cx.Editor.ResetCount()
}

func (e *EditorComponent) replayMacro(
	cx *Context, keys []command.KeyEvent, n int,
) {
	for range n {
		var pending []command.KeyEvent
		i := 0
		for i < len(keys) {
			k := keys[i]
			i++
			mode := cx.Editor.Mode()
			if replaySkip(k, mode) {
				pending = nil
				continue
			}
			if countable(mode, k) {
				ch := k.Code.Char
				cur := cx.Editor.Count()
				if ch >= '1' && ch <= '9' || (ch == '0' && cur > 0) {
					cx.Editor.SetCount(cur*10 + int(ch-'0'))
					continue
				}
			}
			pending = append(pending, k)
			lookup, ok := cx.Keymaps.Lookup(mode, pending)
			if ok && !lookup.Enabled(cx.Editor) {
				ok = false
			}
			switch {
			case ok:
				pending = nil
				cont := lookup.Action(cx.Editor).Continuation
				for cont != nil && i < len(keys) {
					k = keys[i]
					i++
					cont = cont(cx.Editor, k)
				}
				cx.Editor.ResetCount()
			case lookup.Prefix:
				if mode == view.ModeInsert && len(pending) == 1 &&
					pending[0].IsTypable() {
					action.InsertChar(cx.Editor, pending[0].Code.Char)
					pending = nil
					cx.Editor.ResetCount()
				}
			default:
				if mode == view.ModeInsert && len(pending) == 1 &&
					pending[0].IsTypable() {
					action.InsertChar(cx.Editor, pending[0].Code.Char)
				}
				pending = nil
				cx.Editor.ResetCount()
			}
		}
	}
}

func (e *EditorComponent) macroElems(
	cx *Context, base tui.Style,
) []statusElem {
	ms := e.macroSlot
	if !ms.recording {
		return nil
	}
	text := fmt.Sprintf("%s %c", i18n.Text(i18n.StatusMacroRecording), ms.reg)
	if e.macroBlink.phase%2 == 1 {
		return []statusElem{statusBadge(
			strings.Repeat(" ", runewidth.StringWidth(text)), base,
		)}
	}
	return []statusElem{statusBadge(
		text, cx.Theme().Get("ui.statusline.macro"),
	)}
}

func macroText(keys []command.KeyEvent) string {
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k.String()
	}
	return strings.Join(parts, " ")
}

func macroKeys(ed *view.Editor, reg rune) []command.KeyEvent {
	text, ok := ed.Registers().First(reg)
	if !ok {
		return nil
	}
	keys, err := command.ParseKeySequence(text)
	if err != nil {
		return nil
	}
	return keys
}

func replaySkip(k command.KeyEvent, mode view.Mode) bool {
	if mode != view.ModeNormal && mode != view.ModeSelect {
		return false
	}
	if k.Mods == command.ModNone {
		switch k.Code.Char {
		case ':', '/', '?', 's', 'S', 'K', '|', '!', '$', 'Q', 'q':
			return true
		}
	}
	if k.Mods == command.ModAlt {
		switch k.Code.Char {
		case '|', '!', 'K':
			return true
		}
	}
	return false
}
