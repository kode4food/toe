package highlight_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/view"
)

func TestTokenizeLog(t *testing.T) {
	t.Run("scopes tagged lines", func(t *testing.T) {
		lines := []string{
			highlight.LogError + highlight.LogSeparator + "bad",
			highlight.LogWarning + highlight.LogSeparator + "iffy",
			highlight.LogInfo + highlight.LogSeparator + "\u00f6k",
			highlight.LogCommand + highlight.LogSeparator + "ran",
			highlight.LogTerminal + highlight.LogSeparator + "beep",
		}
		spans := highlight.Tokenize(core.Source{
			Text: strings.Join(lines, "\n") + "\n",
			Lang: view.MessagesLanguage,
		})
		assert.Equal(t, []highlight.Span{
			{Start: 0, End: 3, Scope: "ui.log.error"},
			{Start: 6, End: 9, Scope: "ui.log.error"},
			{Start: 10, End: 13, Scope: "ui.log.warning"},
			{Start: 16, End: 20, Scope: "ui.log.warning"},
			{Start: 21, End: 24, Scope: "ui.log.info"},
			{Start: 27, End: 29, Scope: "ui.log.info"},
			{Start: 30, End: 33, Scope: "ui.log.command"},
			{Start: 36, End: 39, Scope: "ui.log.command"},
			{Start: 40, End: 43, Scope: "ui.log.terminal"},
			{Start: 46, End: 50, Scope: "ui.log.terminal"},
		}, spans)
	})

	t.Run("leaves untagged lines alone", func(t *testing.T) {
		spans := highlight.Tokenize(core.Source{
			Text: "plain\nNOT" + highlight.LogSeparator + "a tag\n",
			Lang: view.MessagesLanguage,
		})
		assert.Empty(t, spans)
	})
}
