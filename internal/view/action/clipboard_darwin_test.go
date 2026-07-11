//go:build darwin

package action_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view/action"
)

func TestSystemClipboardDarwin(t *testing.T) {
	t.Run("roundtrips through pasteboard", func(t *testing.T) {
		clip := action.NewSystemClipboard()

		assert.True(t, clip.Available())
		assert.NoError(t, clip.Write("hello from toe"))
		got, err := clip.Read()
		assert.NoError(t, err)
		assert.Equal(t, "hello from toe", got)
	})
}
