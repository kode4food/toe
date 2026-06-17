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
	"github.com/kode4food/toe/internal/view/config"
)

type (
	Report []Check

	Check struct {
		Name    string
		OK      bool
		Detail  string
		Errors  []string
		Warning []string
	}
)

var (
	ErrFailed = errors.New("health check failed")
)

func Run(w io.Writer) error {
	rep := CheckRuntime()
	writeReport(w, rep)
	if !rep.OK() {
		return ErrFailed
	}
	return nil
}

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
		for _, msg := range c.Warning {
			_, _ = fmt.Fprintf(w, "  warning: %s\n", msg)
		}
	}
}

func checkLanguages() Check {
	langs, ok := config.LoadBundledLanguages()
	if !ok {
		return failed("languages", "bundled languages.toml did not parse")
	}
	names := languageNames(langs.Languages)
	errs := compareNames(expectedLanguages(), names)
	return Check{
		Name:   "languages",
		OK:     len(errs) == 0,
		Detail: fmt.Sprintf("%d supported", len(names)),
		Errors: errs,
	}
}

func checkGrammars() Check {
	langs, ok := config.LoadBundledLanguages()
	if !ok {
		return failed("grammars", "bundled languages.toml did not parse")
	}
	names := grammarNames(langs.Grammars)
	errs := compareNames(expectedGrammars(), names)
	return Check{
		Name:   "grammars",
		OK:     len(errs) == 0,
		Detail: fmt.Sprintf("%d configured", len(names)),
		Errors: errs,
	}
}

func checkThemes() Check {
	names := loader.ThemeNames()
	errs := compareNames(expectedThemes(), names)
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

func languageNames(langs []config.Language) []string {
	names := make([]string, 0, len(langs))
	for _, l := range langs {
		names = append(names, l.Name)
	}
	slices.Sort(names)
	return names
}

func grammarNames(grams []config.Grammar) []string {
	names := make([]string, 0, len(grams))
	for _, g := range grams {
		names = append(names, g.Name)
	}
	slices.Sort(names)
	return names
}

func compareNames(expected, actual []string) []string {
	var errs []string
	for _, name := range expected {
		if !slices.Contains(actual, name) {
			errs = append(errs, fmt.Sprintf("missing %s", name))
		}
	}
	for _, name := range actual {
		if !slices.Contains(expected, name) {
			errs = append(errs, fmt.Sprintf("unexpected %s", name))
		}
	}
	return errs
}

func expectedLanguages() []string {
	return []string{
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
}

func expectedGrammars() []string {
	return []string{
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
}

func expectedThemes() []string {
	return []string{
		"frappe",
		"latte",
		"macchiato",
		"mocha",
	}
}
