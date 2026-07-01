package ui

import (
	"errors"
	"fmt"
	"strings"
)

type (
	CompletionOptions struct {
		Icons CompletionIconMode `toml:"icons"`
	}

	CompletionIconMode string
)

const (
	CompletionIconsCodicon CompletionIconMode = "codicon"
	CompletionIconsASCII   CompletionIconMode = "ascii"
	CompletionIconsNone    CompletionIconMode = "none"
)

var (
	ErrCompletionIconMode = errors.New("invalid completion icon mode")
)

func DefaultCompletionOptions() CompletionOptions {
	return CompletionOptions{Icons: CompletionIconsCodicon}
}

func (o CompletionOptions) WithDefaults() CompletionOptions {
	if o.Icons == "" {
		o.Icons = CompletionIconsCodicon
	}
	return o
}

func (m *CompletionIconMode) UnmarshalText(text []byte) error {
	mode := CompletionIconMode(strings.TrimSpace(string(text)))
	if mode == "" {
		*m = CompletionIconsCodicon
		return nil
	}
	if !completionIconModeValid(mode) {
		return fmt.Errorf("%w: %s", ErrCompletionIconMode, mode)
	}
	*m = mode
	return nil
}

func completionIconModeValid(mode CompletionIconMode) bool {
	switch mode {
	case CompletionIconsCodicon, CompletionIconsASCII, CompletionIconsNone:
		return true
	default:
		return false
	}
}
