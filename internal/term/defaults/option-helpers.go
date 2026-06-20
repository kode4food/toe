package defaults

import (
	"runtime"
	"strconv"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/config"
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

func stringOr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func lineNumberOr(
	v view.LineNumber, def view.LineNumber,
) view.LineNumber {
	if v == "" {
		return def
	}
	return v
}

func bufferLineOr(
	v view.BufferLine, def view.BufferLine,
) view.BufferLine {
	if v == "" {
		return def
	}
	return v
}

func editorNullableIntOption(
	key string, dflt int, get func(*view.Editor) *int,
	set func(*view.Editor, *int),
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
	key string, get func(*view.Editor) bool, set func(*view.Editor, bool),
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

func defaultShell() []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/C"}
	}
	return []string{"sh", "-c"}
}
