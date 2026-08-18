package ui

import "os"

// an empty value matches any non-empty value for env
type termMatch struct {
	env   string
	value string
}

func matchesTerm(terms []termMatch) bool {
	for _, t := range terms {
		v := os.Getenv(t.env)
		if v == "" {
			continue
		}
		if t.value == "" || t.value == v {
			return true
		}
	}
	return false
}
