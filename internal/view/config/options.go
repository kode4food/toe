package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/kode4food/toe/internal/toml"
)

var (
	ErrInvalidOption = errors.New("invalid option")
)

// ParseBool parses an option's boolean value
func ParseBool(value string) (bool, error) {
	if v, err := strconv.ParseBool(value); err == nil {
		return v, nil
	}
	return false, fmt.Errorf("%w: %s", ErrInvalidOption, value)
}

// ParsePositiveInt parses an option value that must exceed zero
func ParsePositiveInt(value string) (int, error) {
	v, err := strconv.Atoi(value)
	if err != nil || v < 1 {
		return 0, fmt.Errorf("%w: %s", ErrInvalidOption, value)
	}
	return v, nil
}

// ParseNonNegInt parses an option value that may be zero
func ParseNonNegInt(value string) (int, error) {
	v, err := strconv.Atoi(value)
	if err != nil || v < 0 {
		return 0, fmt.Errorf("%w: %s", ErrInvalidOption, value)
	}
	return v, nil
}

// ParseIntSlice parses a bracketed list of integers
func ParseIntSlice(value string) ([]int, error) {
	var raw struct {
		Value []int `toml:"value"`
	}
	if err := toml.Decode("value = "+value, &raw); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidOption, value)
	}
	return raw.Value, nil
}

// ParseStringSlice parses a bracketed list of strings
func ParseStringSlice(value string) ([]string, error) {
	var raw struct {
		Value []string `toml:"value"`
	}
	if err := toml.Decode("value = "+value, &raw); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidOption, value)
	}
	return raw.Value, nil
}

// ParseStringLiteral parses an optionally quoted string value
func ParseStringLiteral(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	switch value[0] {
	case '"', '\'':
		var raw struct {
			Value string `toml:"value"`
		}
		if err := toml.Decode("value = "+value, &raw); err != nil {
			return "", fmt.Errorf("%w: %s", ErrInvalidOption, value)
		}
		return raw.Value, nil
	default:
		return value, nil
	}
}

// FormatIntSlice renders integers as a bracketed list
func FormatIntSlice(values []int) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, strconv.Itoa(v))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// FormatStringSlice renders strings as a bracketed list
func FormatStringSlice(values []string) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, strconv.Quote(v))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
