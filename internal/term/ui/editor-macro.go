package ui

import (
	"fmt"
	"slices"
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
	return command.ReadChar(func(_ *view.Editor, ch rune) command.Continuation {
		ms.recording = true
		ms.reg = ch
		ms.keys = nil
		return nil
	})
}

// MacroReplayAction prompts for a register key and replays the macro stored
// there count times
func (e *EditorComponent) MacroReplayAction(
	_ *view.Editor,
) command.Continuation {
	ms := e.macroSlot
	if ms.recording {
		return nil
	}
	return command.ReadChar(func(
		ed *view.Editor, ch rune,
	) command.Continuation {
		ms.replayReg = ch
		ms.replayCount = ed.CountOr(1)
		ms.hasReplay = true
		return nil
	})
}

func (e *EditorComponent) handleReplay(cx *Context) {
	ms := e.macroSlot
	if !ms.hasReplay {
		return
	}
	ms.hasReplay = false
	e.replayMacro(cx, macroKeys(cx.Editor, ms.replayReg), ms.replayCount)
	cx.Editor.SetCount(0)
}

func (e *EditorComponent) replayMacro(
	cx *Context, keys []command.KeyEvent, n int,
) {
	for range n {
		var pending []command.KeyEvent
		count := 0
		i := 0
		for i < len(keys) {
			k := keys[i]
			i++
			mode := cx.Editor.Mode()
			if replaySkip(k, mode) {
				pending = nil
				continue
			}
			if countable(mode, k) && cx.Keymaps.AcceptsCount(mode, pending) {
				ch := k.Code.Char
				cur := count
				if ch >= '1' && ch <= '9' || (ch == '0' && cur > 0) {
					count = cur*10 + int(ch-'0')
					cx.Editor.SetCount(count)
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
				seq := pending
				pending = nil
				cont := lookup.Action(cx.Editor).Continuation
				var frames []command.Continuation
				popped := false
				for cont != nil && i < len(keys) {
					k = keys[i]
					i++
					next, step := cont(cx.Editor, k)
					switch step {
					case command.ContinuationStay:
						if next != nil {
							cont = next
						}
					case command.ContinuationPush:
						frames = append(frames, cont)
						seq = append(seq, k)
						cont = next
					case command.ContinuationPop:
						if n := len(frames); n > 0 {
							cont = frames[n-1]
							frames = frames[:n-1]
							seq = seq[:len(seq)-1]
							continue
						}
						pending = slices.Clone(seq[:len(seq)-1])
						popped = true
						cont = nil
					default:
						cont = nil
					}
				}
				if !popped {
					count = 0
					cx.Editor.SetCount(0)
				}
			case lookup.Prefix:
				if mode == view.ModeInsert && len(pending) == 1 &&
					pending[0].IsTypable() {
					action.InsertChar(cx.Editor, pending[0].Code.Char)
					pending = nil
					count = 0
					cx.Editor.SetCount(0)
				}
			default:
				if mode == view.ModeInsert && len(pending) == 1 &&
					pending[0].IsTypable() {
					action.InsertChar(cx.Editor, pending[0].Code.Char)
				}
				pending = nil
				count = 0
				cx.Editor.SetCount(0)
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
