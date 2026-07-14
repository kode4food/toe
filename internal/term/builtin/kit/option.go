package kit

import (
	"strconv"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/config"
)

type (
	// EditorGetter reads a typed value from the editor
	EditorGetter[T any] func(*view.Editor) T

	// EditorSetter writes a typed value to the editor
	EditorSetter[T any] func(*view.Editor, T)
)

// BoolOr dereferences p, falling back to def when p is nil
func BoolOr(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

// IntOr dereferences p, falling back to def when p is nil
func IntOr(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

// EditorNullableIntOption builds an option over a nullable int editor value,
// reporting dflt when unset
func EditorNullableIntOption(
	key string, dflt int, get EditorGetter[*int], set EditorSetter[*int],
) command.Option {
	return command.Option{
		Key: key,
		Get: func(e *view.Editor) (string, error) {
			if p := get(e); p != nil {
				return strconv.Itoa(*p), nil
			}
			return strconv.Itoa(dflt), nil
		},
		Set: func(e *view.Editor, s string) error {
			v, err := config.ParsePositiveInt(s)
			if err != nil {
				return err
			}
			set(e, &v)
			return nil
		},
	}
}

// EditorBoolOption builds a toggleable option over a bool editor value
func EditorBoolOption(
	key string, get EditorGetter[bool], set EditorSetter[bool],
) command.Option {
	return command.Option{
		Key: key,
		Get: func(e *view.Editor) (string, error) {
			return strconv.FormatBool(get(e)), nil
		},
		Set: func(e *view.Editor, s string) error {
			v, err := config.ParseBool(s)
			if err != nil {
				return err
			}
			set(e, v)
			return nil
		},
		Toggle: func(e *view.Editor) (string, error) {
			v := !get(e)
			set(e, v)
			return strconv.FormatBool(v), nil
		},
	}
}
