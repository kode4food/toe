package loader

import "errors"

var (
	ErrPathUnavailable = errors.New("path unavailable")
	ErrThemeNotFound   = errors.New("theme not found")
	ErrThemeCycle      = errors.New("theme inheritance cycle")
)
