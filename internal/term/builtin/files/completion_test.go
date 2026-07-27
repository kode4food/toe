package files_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin/test"
)

func TestCompletionOptions(t *testing.T) {
	cases := []struct{ key, value string }{
		{key: "completion.auto", value: "false"},
		{key: "completion.delay", value: "0"},
		{key: "completion.trigger-len", value: "3"},
	}
	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			e, km := test.Env(t, "")
			test.RunCmdArgs(t,
				km, e, "set-option", tc.key+" "+tc.value,
			)
			res := test.RunCmdArgs(t, km, e, "get-option", tc.key)
			assert.Equal(t, tc.value, res.Message)
		})
	}

	t.Run("defaults", func(t *testing.T) {
		e, km := test.Env(t, "")
		for key, want := range map[string]string{
			"completion.auto":        "true",
			"completion.delay":       "250",
			"completion.trigger-len": "2",
		} {
			res := test.RunCmdArgs(t, km, e, "get-option", key)
			assert.Equal(t, want, res.Message)
		}
	})
}
