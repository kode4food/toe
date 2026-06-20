package language

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
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
			if ft.Glob != "" && globMatch(ft.Glob, abs) {
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

func languageRootMarkerExists(dir string, roots []string) bool {
	osEntries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	entries := make([]string, 0, len(osEntries))
	for _, e := range osEntries {
		entries = append(entries, e.Name())
	}
	for _, name := range entries {
		for _, root := range roots {
			if rootMarkerMatches(root, name) {
				return true
			}
		}
	}
	return false
}

func rootMarkerMatches(pattern, name string) bool {
	return globMatch(pattern, name) || pattern == name
}

func globMatch(pattern, path string) bool {
	for _, pattern := range expandGlobBraces(pattern) {
		if ok := pathGlobMatch(pattern, filepath.ToSlash(path)); ok {
			return true
		}
		if pathGlobMatch(pattern, path) {
			return true
		}
	}
	return false
}

func pathGlobMatch(pattern, path string) bool {
	parts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")
	if strings.HasPrefix(pattern, "*/") {
		for i := range len(pathParts) {
			if pathGlobParts(parts, pathParts[i:]) {
				return true
			}
		}
	}
	return pathGlobParts(parts, pathParts)
}

func pathGlobParts(pattern, path []string) bool {
	for len(pattern) > 0 {
		part := pattern[0]
		pattern = pattern[1:]
		if part == "**" {
			if len(pattern) == 0 {
				return true
			}
			for i := range len(path) + 1 {
				if pathGlobParts(pattern, path[i:]) {
					return true
				}
			}
			return false
		}
		if len(path) == 0 {
			return false
		}
		if ok, err := filepath.Match(part, path[0]); err != nil || !ok {
			return false
		}
		path = path[1:]
	}
	return len(path) == 0
}

func expandGlobBraces(pattern string) []string {
	start := strings.Index(pattern, "{")
	if start < 0 {
		return []string{pattern}
	}
	end := strings.Index(pattern[start:], "}")
	if end < 0 {
		return []string{pattern}
	}
	end += start
	pfx := pattern[:start]
	sfx := pattern[end+1:]
	alts := strings.Split(pattern[start+1:end], ",")
	out := make([]string, 0, len(alts))
	for _, alt := range alts {
		out = append(out, expandGlobBraces(pfx+alt+sfx)...)
	}
	return out
}

func normalizeGlob(glob string) string {
	if filepath.IsAbs(glob) || strings.HasPrefix(glob, "*/") {
		return glob
	}
	return "*/" + glob
}
