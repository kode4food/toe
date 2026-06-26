//go:build !windows

package view

// DefaultShell returns the platform shell command prefix for shell actions
func DefaultShell() []string {
	return []string{"sh", "-c"}
}
