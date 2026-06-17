package ui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

var (
	ErrNoFocusedDocument = errors.New("no focused document")
	ErrNoFocusedView     = errors.New("no focused view")
	ErrUnknownVariable   = errors.New("unknown variable")
	ErrInvalidUnicode    = errors.New("invalid Unicode codepoint")
	ErrShellExpansion    = errors.New("shell expansion failed")
	ErrInvalidRegister   = errors.New("invalid register")
)

// NewTokenExpander returns a TokenExpander that resolves percent-expansions
// using the current editor state (selections, registers, variables, shell)
func NewTokenExpander(e *view.Editor) command.TokenExpander {
	return func(tok command.Token) (string, error) {
		switch tok.Kind {
		case command.TokenUnquoted, command.TokenQuoted:
			return tok.Content, nil
		case command.TokenExpansion:
			return expandExpansion(e, tok)
		case command.TokenExpand:
			return expandInner(e, tok.Content)
		default:
			return tok.Content, nil
		}
	}
}

func expandExpansion(e *view.Editor, tok command.Token) (string, error) {
	switch tok.Expansion {
	case command.ExpansionVariable:
		return expandVariable(e, tok.Content)
	case command.ExpansionUnicode:
		return expandUnicode(tok.Content)
	case command.ExpansionShell:
		inner, err := expandInner(e, tok.Content)
		if err != nil {
			return "", err
		}
		return expandShell(e, inner)
	case command.ExpansionRegister:
		return expandRegister(e, tok.Content)
	}
	return tok.Content, nil
}

func expandVariable(e *view.Editor, name string) (string, error) {
	doc, ok := e.FocusedDocument()
	if !ok {
		return "", ErrNoFocusedDocument
	}
	v, ok := e.FocusedView()
	if !ok {
		return "", ErrNoFocusedView
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	prim := sel.Primary()
	cursor := prim.Cursor(text)

	switch name {
	case "cursor_line":
		line, _ := text.CharToLine(cursor)
		return fmt.Sprintf("%d", line+1), nil

	case "cursor_column":
		line, _ := text.CharToLine(cursor)
		lineStart, _ := text.LineToChar(line)
		return fmt.Sprintf("%d", cursor-lineStart+1), nil

	case "buffer_name":
		if p := doc.Path(); p != "" {
			if rel, err := filepath.Rel(e.Cwd(), p); err == nil {
				return rel, nil
			}
			return p, nil
		}
		return view.ScratchBufferName, nil

	case "file_path_absolute":
		if p := doc.Path(); p != "" {
			if abs, err := filepath.Abs(p); err == nil {
				return abs, nil
			}
			return p, nil
		}
		return e.Cwd(), nil

	case "line_ending":
		return string(doc.LineEnding()), nil

	case "current_working_directory":
		return e.Cwd(), nil

	case "workspace_directory":
		return findWorkspace(e.Cwd()), nil

	case "language":
		if lang := doc.Lang(); lang != "" && lang != "text" {
			return lang, nil
		}
		return "text", nil

	case "selection":
		if sl, err := text.Slice(prim.From(), prim.To()); err == nil {
			return sl.String(), nil
		}
		return "", nil

	case "selection_line_start":
		line, _ := text.CharToLine(prim.From())
		return fmt.Sprintf("%d", line+1), nil

	case "selection_line_end":
		line, _ := text.CharToLine(prim.To())
		return fmt.Sprintf("%d", line+1), nil
	}
	return "", fmt.Errorf("%w: %s", ErrUnknownVariable, name)
}

func expandUnicode(content string) (string, error) {
	var codepoint uint32
	_, err := fmt.Sscanf(content, "%x", &codepoint)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrInvalidUnicode, content)
	}
	r := rune(codepoint)
	if !utf8.ValidRune(r) {
		return "", fmt.Errorf("%w: %s", ErrInvalidUnicode, content)
	}
	return string(r), nil
}

func expandShell(e *view.Editor, content string) (string, error) {
	shell := e.Config().Shell()
	if len(shell) == 0 {
		shell = defaultShell()
	}
	args := append(shell[1:len(shell):len(shell)], content)
	cmd := exec.Command(shell[0], args...)
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		if len(out) == 0 {
			return "", fmt.Errorf("%w: %w", ErrShellExpansion, err)
		}
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

func expandRegister(e *view.Editor, content string) (string, error) {
	runes := []rune(content)
	if len(runes) != 1 {
		return "", fmt.Errorf("%w: %s", ErrInvalidRegister, content)
	}
	vals := e.Registers().Read(runes[0])
	return strings.Join(vals, "\n"), nil
}

func expandInner(e *view.Editor, content string) (string, error) {
	var buf strings.Builder
	start := 0
	for {
		idx := strings.IndexByte(content[start:], '%')
		if idx < 0 {
			break
		}
		idx += start
		buf.WriteString(content[start:idx])

		if idx+1 < len(content) && content[idx+1] == '%' {
			buf.WriteByte('%')
			start = idx + 2
			continue
		}

		t := command.NewTokenizer(content[idx:], true)
		tok, ok, err := t.Next()
		if err != nil || !ok {
			buf.WriteString(content[idx:])
			start = len(content)
			break
		}
		expanded, err := expandExpansion(e, tok)
		if err != nil {
			return "", err
		}
		buf.WriteString(expanded)
		start = idx + t.Pos()
	}
	buf.WriteString(content[start:])
	return buf.String(), nil
}

func findWorkspace(dir string) string {
	markers := []string{".git", ".svn", ".toe", "jj"}
	d := dir
	for {
		for _, m := range markers {
			if _, err := os.Stat(filepath.Join(d, m)); err == nil {
				return d
			}
		}
		parent := filepath.Dir(d)
		if parent == d {
			break
		}
		d = parent
	}
	return dir
}

func defaultShell() []string {
	if shell := os.Getenv("SHELL"); shell != "" {
		return []string{shell, "-c"}
	}
	return []string{"sh", "-c"}
}
