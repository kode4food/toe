package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInsertRegister(t *testing.T) {
	t.Run("insert register takes a char", func(t *testing.T) {
		e, km := defaultsEnv(t, "abc")
		setCursor(t, e, 0)
		res := runCmd(t, km, e, "insert_register")
		assert.NotNil(t, res.Continuation)
		// empty register pastes nothing; the continuation still completes
		assert.Nil(t, res.Continuation(e, char('a')))
	})
}
