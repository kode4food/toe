package defaults

import (
	"strconv"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/config"
)

type (
	editorGetter[T any] func(*view.Editor) T
	editorSetter[T any] func(*view.Editor, T)
)

func boolOr(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func intOr(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

func editorNullableIntOption(
	key string, dflt int, get editorGetter[*int], set editorSetter[*int],
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

func editorBoolOption(
	key string, get editorGetter[bool], set editorSetter[bool],
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
