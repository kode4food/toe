package glob

import (
	"path/filepath"
	"strings"
)

// Candidate is a glob pattern and the path being tested against it
type Candidate struct {
	Pattern string
	Path    string
}

// Match reports whether Path matches the glob Pattern, expanding brace
// alternatives and matching with both native and slash separators
func Match(c Candidate) bool {
	path := c.Path
	for _, p := range expandBraces(c.Pattern) {
		if matchPath(Candidate{Pattern: p, Path: filepath.ToSlash(path)}) {
			return true
		}
		if matchPath(Candidate{Pattern: p, Path: path}) {
			return true
		}
	}
	return false
}

func expandBraces(pattern string) []string {
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
		out = append(out, expandBraces(pfx+alt+sfx)...)
	}
	return out
}

func matchPath(c Candidate) bool {
	parts := strings.Split(c.Pattern, "/")
	pathParts := strings.Split(c.Path, "/")
	if strings.HasPrefix(c.Pattern, "*/") {
		for i := range len(pathParts) {
			if matchParts(matchPartsArgs{
				pattern: parts,
				path:    pathParts[i:],
			}) {
				return true
			}
		}
	}
	return matchParts(matchPartsArgs{pattern: parts, path: pathParts})
}

// matchPartsArgs is a Candidate already split on slashes
type matchPartsArgs struct {
	pattern []string
	path    []string
}

func matchParts(c matchPartsArgs) bool {
	pattern := c.pattern
	path := c.path
	for len(pattern) > 0 {
		part := pattern[0]
		pattern = pattern[1:]
		if part == "**" {
			if len(pattern) == 0 {
				return true
			}
			for i := range len(path) + 1 {
				if matchParts(matchPartsArgs{
					pattern: pattern,
					path:    path[i:],
				}) {
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
