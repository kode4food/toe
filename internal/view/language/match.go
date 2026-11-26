package language

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/kode4food/toe/internal/glob"
)

// ForFilename returns the configured language matching path
func ForFilename(langs Languages, path string) *Language {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	var found *Language
	foundLen := -1
	for i, lang := range langs.Languages {
		for _, ft := range lang.FileTypes {
			if ft.Glob != "" && glob.Match(glob.Candidate{
				Pattern: ft.Glob,
				Path:    abs,
			}) {
				if n := len(ft.Glob); n > foundLen {
					found = &langs.Languages[i]
					foundLen = n
				}
			}
		}
	}
	if foundLen >= 0 {
		return found
	}
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	name := filepath.Base(path)
	for i, lang := range langs.Languages {
		for _, ft := range lang.FileTypes {
			if ext != "" && ft.Extension == ext {
				return &langs.Languages[i]
			}
			if ft.Extension == name {
				return &langs.Languages[i]
			}
		}
	}
	return nil
}

func languageForShebang(langs Languages, content string) (string, bool) {
	line := content
	if before, _, ok := strings.Cut(content, "\n"); ok {
		line = before
	}
	if !strings.HasPrefix(line, "#!") {
		return "", false
	}
	fields := strings.Fields(strings.TrimPrefix(line, "#!"))
	if len(fields) == 0 {
		return "", false
	}
	marker := filepath.Base(fields[0])
	if marker == "env" && len(fields) > 1 {
		marker = fields[len(fields)-1]
	}
	for _, lang := range langs.Languages {
		if slices.Contains(lang.Shebangs, marker) {
			return lang.Name, true
		}
	}
	return "", false
}
