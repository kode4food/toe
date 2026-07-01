package language

import (
	"path/filepath"
	"regexp"

	"slices"
	"strings"

	"github.com/kode4food/toe/internal/glob"
)

func languageForFilename(langs Languages, path string) (Language, bool) {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	var found Language
	foundLen := -1
	for _, lang := range langs.Languages {
		for _, ft := range lang.FileTypes {
			if ft.Glob != "" && glob.Match(ft.Glob, abs) {
				if n := len(ft.Glob); n > foundLen {
					found = lang
					foundLen = n
				}
			}
		}
	}
	if foundLen >= 0 {
		return found, true
	}
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	name := filepath.Base(path)
	for _, lang := range langs.Languages {
		for _, ft := range lang.FileTypes {
			if ext != "" && ft.Extension == ext {
				return lang, true
			}
			if ft.Extension == name {
				return lang, true
			}
		}
	}
	return Language{}, false
}

func languageForMatch(langs Languages, text string) (string, bool) {
	for _, lang := range langs.Languages {
		if lang.Name == text {
			return lang.Name, true
		}
	}
	bestLen := 0
	var found string
	for _, lang := range langs.Languages {
		if lang.InjectionRegex == "" {
			continue
		}
		re, err := regexp.Compile(lang.InjectionRegex)
		if err != nil {
			continue
		}
		loc := re.FindStringIndex(text)
		if loc == nil {
			continue
		}
		if n := loc[1] - loc[0]; n > bestLen {
			bestLen = n
			found = lang.Name
		}
	}
	return found, found != ""
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
