// Package vcs integrates version-control systems with the editor: diff bases
// for gutter hunks, changed-file listings, and head names. Git is the only
// provider today; the Provider interface keeps additional systems pluggable
package vcs

import "github.com/kode4food/toe/internal/view"

type (
	// Registry queries all active providers in order; the first provider that
	// answers a request wins
	Registry struct {
		providers []Provider
	}

	// Provider supplies diff bases and change information from one
	// version-control system
	Provider interface {
		// DiffBase returns the checked-in contents of path, the "base" text a
		// diff of the working copy is computed against
		DiffBase(path string) ([]byte, error)

		// HeadName returns a short display name for the current head, such as a
		// branch name
		HeadName(path string) (string, error)

		// ChangedFiles lists workspace files that differ from the head
		ChangedFiles(cwd string) ([]view.FileChange, error)
	}
)

// NewRegistry returns a registry over all supported providers
func NewRegistry() *Registry {
	return &Registry{providers: []Provider{Git{}}}
}

// DiffBase returns the diff base for path from the first provider that has one
func (r *Registry) DiffBase(path string) ([]byte, bool) {
	for _, p := range r.providers {
		if base, err := p.DiffBase(path); err == nil {
			return base, true
		}
	}
	return nil, false
}

// HeadName returns the head name for path from the first provider that has one
func (r *Registry) HeadName(path string) (string, bool) {
	for _, p := range r.providers {
		if name, err := p.HeadName(path); err == nil {
			return name, true
		}
	}
	return "", false
}

// ChangedFiles lists changed files under cwd from the first provider that
// reports success
func (r *Registry) ChangedFiles(cwd string) ([]view.FileChange, bool) {
	for _, p := range r.providers {
		if changes, err := p.ChangedFiles(cwd); err == nil {
			return changes, true
		}
	}
	return nil, false
}
