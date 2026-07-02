package glob_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/glob"
)

func TestGlob(t *testing.T) {
	t.Run("expands braces", func(t *testing.T) {
		assert.Equal(t,
			[]string{"*.go", "*.mod", "*.sum"},
			glob.ExpandBraces("*.{go,mod,sum}"),
		)
		assert.Equal(t,
			[]string{
				"cmd/a.go", "cmd/a.txt", "internal/a.go", "internal/a.txt",
			},
			glob.ExpandBraces("{cmd,internal}/a.{go,txt}"),
		)
	})

	t.Run("keeps malformed braces", func(t *testing.T) {
		assert.Equal(t, []string{"*.{go"}, glob.ExpandBraces("*.{go"))
		assert.Equal(t, []string{"*.go"}, glob.ExpandBraces("*.go"))
	})

	t.Run("matches paths", func(t *testing.T) {
		cases := []struct {
			name    string
			pattern string
			path    string
			want    bool
		}{
			{
				name:    "literal",
				pattern: "cmd/toe",
				path:    "cmd/toe",
				want:    true,
			},
			{
				name:    "brace",
				pattern: "*.{go,mod}",
				path:    "main.go",
				want:    true,
			},
			{
				name:    "prefix",
				pattern: "*/main.go",
				path:    "cmd/toe/main.go",
				want:    true,
			},
			{
				name:    "globstar",
				pattern: "**/*.go",
				path:    "a/b/main.go",
				want:    true,
			},
			{
				name:    "tail globstar",
				pattern: "cmd/**",
				path:    "cmd/toe/main.go",
				want:    true,
			},
			{
				name:    "globstar miss",
				pattern: "**/*.go",
				path:    "a/b/main.txt",
				want:    false,
			},
			{
				name:    "path too short",
				pattern: "cmd/toe/*.go",
				path:    "cmd/toe",
				want:    false,
			},
			{
				name:    "native",
				pattern: "cmd/toe/*.go",
				path:    "cmd/toe/main.go",
				want:    true,
			},
			{
				name:    "mismatch",
				pattern: "cmd/*.go",
				path:    "cmd/toe/main.go",
				want:    false,
			},
			{
				name:    "bad pattern",
				pattern: "[",
				path:    "main.go",
				want:    false,
			},
		}

		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.want, glob.Match(tt.pattern, tt.path))
			})
		}
	})
}
