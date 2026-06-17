package core_test

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
)

func TestRopeReader(t *testing.T) {
	t.Run("reads rope bytes", func(t *testing.T) {
		r := core.NewRopeReader(core.NewRope("a世界"))
		buf := make([]byte, 4)

		n, err := r.Read(buf)

		assert.NoError(t, err)
		assert.Equal(t, 4, n)
		assert.Equal(t, "a世", string(buf))
	})

	t.Run("continues until EOF", func(t *testing.T) {
		r := core.NewRopeReader(core.NewRope("abc"))
		buf := make([]byte, 2)

		n, err := r.Read(buf)
		assert.NoError(t, err)
		assert.Equal(t, 2, n)
		assert.Equal(t, "ab", string(buf))

		n, err = r.Read(buf)
		assert.NoError(t, err)
		assert.Equal(t, 1, n)
		assert.Equal(t, "c", string(buf[:n]))

		n, err = r.Read(buf)
		assert.True(t, errors.Is(err, io.EOF))
		assert.Equal(t, 0, n)
	})
}
