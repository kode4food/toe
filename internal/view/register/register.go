// Package register implements the editor register store
package register

// Registers is the key-value store that maps register names to value lists
// Special registers:
//   - '_' (black hole): all reads return empty; writes are discarded
//   - '"' (default): the default yank register
type Registers map[rune][]string

// New returns an empty Registers
func New() Registers {
	return make(Registers)
}

// Write stores values under name. For the black-hole register ('_') the write
// is silently discarded
func (r Registers) Write(name rune, values []string) {
	if name == '_' {
		return
	}
	r[name] = values
}

// Read returns the values stored under name, in insertion order. Returns nil
// if the register is empty or unset
func (r Registers) Read(name rune) []string {
	if name == '_' {
		return nil
	}
	vals, ok := r[name]
	if !ok || len(vals) == 0 {
		return nil
	}
	return vals
}

// First returns the first value stored under name, or ("", false)
func (r Registers) First(name rune) (string, bool) {
	vals := r.Read(name)
	if len(vals) == 0 {
		return "", false
	}
	return vals[0], true
}

// Set stores a single string value under name
func (r Registers) Set(name rune, value string) {
	r.Write(name, []string{value})
}

// Clear removes all values stored under name
func (r Registers) Clear(name rune) {
	delete(r, name)
}

// ClearAll removes all register values
func (r Registers) ClearAll() {
	clear(r)
}
