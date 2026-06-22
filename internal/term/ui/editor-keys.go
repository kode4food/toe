package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	act "github.com/kode4food/toe/internal/view/action"
)

func (e *EditorComponent) handleKeyPress(
	msg tea.KeyPressMsg, cx *Context,
) (EventResult, tea.Cmd) {
	k := FromTeaKey(msg)

	if len(e.pending) == 0 && k.Code.Char == 'z' && k.Mods == command.ModCtrl {
		return consumed(), tea.Suspend
	}

	if e.macroSlot.recording {
		e.macroSlot.keys = append(e.macroSlot.keys, k)
	}

	if e.continuation != nil {
		cont := e.continuation(cx.Editor, k)
		e.continuation = cont
		if cont == nil {
			e.hint = ""
			cx.Editor.ResetCount()
		}
		e.syncEditorMessages(cx)
		e.handleReplay(cx)
		return consumed(), nil
	}

	mode := cx.Editor.Mode()
	modeStr := mode.String()

	noPending := len(e.pending) == 0 && k.Mods == command.ModNone &&
		k.Code.Special == ""
	if noPending && mode == view.ModeSelect {
		ch := k.Code.Char
		cur := cx.Editor.Count()
		if ch >= '1' && ch <= '9' || (ch == '0' && cur > 0) {
			cx.Editor.SetCount(cur*10 + int(ch-'0'))
			e.status = fmt.Sprintf("%d", cx.Editor.Count())
			return consumed(), nil
		}
	}

	e.pending = append(e.pending, k)
	action, found, prefix := cx.Keymaps.Lookup(modeStr, e.pending)
	switch {
	case found:
		e.pending = nil
		e.status = ""
		e.cmdMsg = ""
		e.infoTitle = ""
		e.infoItems = nil
		cont := action(cx.Editor)
		e.continuation = cont
		if cont == nil {
			e.hint = ""
			cx.Editor.ResetCount()
		}
		e.syncEditorMessages(cx)
		e.handleReplay(cx)

		if ov := e.nextLayer; ov != nil {
			e.nextLayer = nil
			return consumedWith(func(comp *Compositor, cx *Context) tea.Cmd {
				layer, cmd := ov(cx)
				if layer != nil {
					comp.Push(layer)
				}
				return cmd
			}), nil
		}
		return consumed(), nil

	case prefix:
		var sb strings.Builder
		if c := cx.Editor.Count(); c > 0 {
			_, _ = fmt.Fprintf(&sb, "%d", c)
		}
		for _, pk := range e.pending {
			sb.WriteString(pk.String())
		}
		e.status = sb.String()
		title, items := cx.Keymaps.PendingHints(modeStr, e.pending)
		e.infoTitle = title
		e.infoItems = items
		return consumed(), nil

	default:
		if mode == view.ModeInsert && len(e.pending) == 1 {
			if e.pending[0].IsTypable() {
				act.InsertChar(cx.Editor, e.pending[0].Code.Char)
			}
		}
		e.pending = nil
		e.status = ""
		e.infoTitle = ""
		e.infoItems = nil
		cx.Editor.ResetCount()
		return consumed(), nil
	}
}
