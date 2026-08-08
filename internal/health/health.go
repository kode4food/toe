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

// CheckRuntime runs every bundled-asset check
func CheckRuntime() Report {
	return Report{
		checkLanguages(),
		checkThemes(),
		checkSyntaxQueries(),
	}
}

// OK reports whether every check passed
func (r Report) OK() bool {
	for _, c := range r {
		if !c.OK {
			return false
		}
	}
	return true
}

// Run writes the runtime report to w, erroring when a check failed
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
	if langs, ok := language.LoadBundledLanguages(); ok {
		return Check{
			Name:   "languages",
			OK:     true,
			Detail: fmt.Sprintf("%d supported", len(langs.Languages)),
		}
	}
	return failed(failedArgs{
		name:    "languages",
		message: "bundled languages.toml did not parse",
	})
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
		th, warnings := theme.Decode(data)
		if err := th.Validate(); err != nil {
			errs = append(errs, fmt.Sprintf("%s is invalid: %v", name, err))
		}
		for _, w := range warnings {
			errs = append(errs, fmt.Sprintf("%s: %s", name, w))
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

type failedArgs struct {
	name    string
	message string
}

func failed(args failedArgs) Check {
	return Check{Name: args.name, Errors: []string{args.message}}
}
