package language

import (
	"path/filepath"
	"strings"
)

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
