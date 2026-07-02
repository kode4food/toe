package health

import (
	"errors"
	"fmt"
	"io"
	"slices"
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

var (
	expectedLanguages = []string{
		"bash",
		"css",
		"dockerfile",
		"env",
		"gitcommit",
		"gitattributes",
		"gitignore",
		"go",
		"gomod",
		"graphql",
		"hcl",
		"html",
		"javascript",
		"json",
		"jsonc",
		"markdown",
		"markdown.inline",
		"protobuf",
		"sql",
		"toml",
		"tsx",
		"typescript",
		"yaml",
	}

	expectedGrammars = []string{
		"bash",
		"css",
		"dockerfile",
		"gitattributes",
		"gitcommit",
		"gitignore",
		"go",
		"gomod",
		"graphql",
		"hcl",
		"html",
		"javascript",
		"json",
		"markdown",
		"markdown_inline",
		"proto",
		"sql",
		"toml",
		"tsx",
		"typescript",
		"yaml",
	}

	expectedThemes = []string{
		"frappe",
		"latte",
		"macchiato",
		"mocha",
	}
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
	names := make([]string, 0, len(langs.Languages))
	for _, l := range langs.Languages {
		names = append(names, l.Name)
	}
	slices.Sort(names)
	errs := compareNames(expectedLanguages, names)
	return Check{
		Name:   "languages",
		OK:     len(errs) == 0,
		Detail: fmt.Sprintf("%d supported", len(names)),
		Errors: errs,
	}
}

func checkGrammars() Check {
	langs, ok := language.LoadBundledLanguages()
	if !ok {
		return failed("grammars", "bundled languages.toml did not parse")
	}
	names := make([]string, 0, len(langs.Grammars))
	for _, g := range langs.Grammars {
		names = append(names, g.Name)
	}
	slices.Sort(names)
	errs := compareNames(expectedGrammars, names)
	return Check{
		Name:   "grammars",
		OK:     len(errs) == 0,
		Detail: fmt.Sprintf("%d configured", len(names)),
		Errors: errs,
	}
}

func checkThemes() Check {
	names := loader.ThemeNames()
	errs := compareNames(expectedThemes, names)
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

func compareNames(expected, actual []string) []string {
	expSet := make(map[string]bool, len(expected))
	for _, name := range expected {
		expSet[name] = true
	}
	actSet := make(map[string]bool, len(actual))
	for _, name := range actual {
		actSet[name] = true
	}
	var errs []string
	for _, name := range expected {
		if !actSet[name] {
			errs = append(errs, fmt.Sprintf("missing %s", name))
		}
	}
	for _, name := range actual {
		if !expSet[name] {
			errs = append(errs, fmt.Sprintf("unexpected %s", name))
		}
	}
	return errs
}
