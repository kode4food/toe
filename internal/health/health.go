package health

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/term/syntax"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/view/language"
)

type (
	Report []Check

	Check struct {
		Name   string
		OK     bool
		Detail string
		Errors []string
	}
)

var (
	ErrFailed = errors.New("health check failed")
)

func CheckRuntime() Report {
	return Report{
		checkLanguages(),
		checkGrammars(),
		checkThemes(),
		checkSyntaxQueries(),
	}
}

func (r Report) OK() bool {
	for _, c := range r {
		if !c.OK {
			return false
		}
	}
	return true
}

func Run(w io.Writer) error {
	rep := CheckRuntime()
	writeReport(w, rep)
	if !rep.OK() {
		return ErrFailed
	}
	return nil
}

func writeReport(w io.Writer, r Report) {
	status := "ok"
	if !r.OK() {
		status = "failed"
	}
	_, _ = fmt.Fprintf(w, "toe health: %s\n", status)
	for _, c := range r {
		mark := "ok"
		if !c.OK {
			mark = "fail"
		}
		_, _ = fmt.Fprintf(w, "- %s: %s", c.Name, mark)
		if c.Detail != "" {
			_, _ = fmt.Fprintf(w, " (%s)", c.Detail)
		}
		_, _ = fmt.Fprintln(w)
		for _, msg := range c.Errors {
			_, _ = fmt.Fprintf(w, "  error: %s\n", msg)
		}
	}
}

func checkLanguages() Check {
	langs, ok := language.LoadBundledLanguages()
	if !ok {
		return failed("languages", "bundled languages.toml did not parse")
	}
	return Check{
		Name:   "languages",
		OK:     true,
		Detail: fmt.Sprintf("%d supported", len(langs.Languages)),
	}
}

func checkGrammars() Check {
	langs, ok := language.LoadBundledLanguages()
	if !ok {
		return failed("grammars", "bundled languages.toml did not parse")
	}
	return Check{
		Name:   "grammars",
		OK:     true,
		Detail: fmt.Sprintf("%d configured", len(langs.Grammars)),
	}
}

func checkThemes() Check {
	names := loader.ThemeNames()
	var errs []string
	for _, name := range names {
		data, err := loader.LoadThemeTOML(name)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s did not load: %v", name, err))
			continue
		}
		th, _ := theme.Decode(data)
		if err := th.Validate(); err != nil {
			errs = append(errs, fmt.Sprintf("%s is invalid: %v", name, err))
		}
	}
	return Check{
		Name:   "themes",
		OK:     len(errs) == 0,
		Detail: strings.Join(names, ", "),
		Errors: errs,
	}
}

func checkSyntaxQueries() Check {
	names := syntax.SupportedLanguages()
	var errs []string
	for _, name := range names {
		if !syntax.HasHighlightQuery(name) {
			errs = append(errs, fmt.Sprintf("%s has no highlight query", name))
		}
	}
	return Check{
		Name:   "syntax queries",
		OK:     len(errs) == 0,
		Detail: fmt.Sprintf("%d highlighters", len(names)),
		Errors: errs,
	}
}

func failed(name, msg string) Check {
	return Check{Name: name, OK: false, Errors: []string{msg}}
}
