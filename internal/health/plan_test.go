package health_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type planCheck struct {
	item  string
	tests []string
}

func TestPlanStatus(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "PLAN.md"))
	assert.NoError(t, err)
	plan := string(data)

	for _, check := range planChecks() {
		t.Run(check.item, func(t *testing.T) {
			assert.True(t, planItemChecked(plan, check.item))
			for _, path := range check.tests {
				_, err := os.Stat(filepath.Join(root, path))
				assert.NoError(t, err)
			}
		})
	}
}

func planChecks() []planCheck {
	return []planCheck{
		{
			item: "Add a check that validates `PLAN.md` status against " +
				"implemented package tests where practical.",
			tests: []string{
				"internal/health/plan_test.go",
			},
		},
		{
			item: "Finish runtime asset layout validation for supported " +
				"languages only.",
			tests: []string{
				"internal/health/health_test.go",
				"internal/loader/toml_test.go",
				"internal/term/syntax/syntax_test.go",
			},
		},
		{
			item: "Diagnostics picker and workspace diagnostics picker.",
			tests: []string{
				"internal/term/ui/picker-diagnostics_test.go",
			},
		},
		{
			item: "Every documented command is registered.",
			tests: []string{
				"internal/term/defaults/register_test.go",
			},
		},
		{
			item: "Every default keybinding resolves.",
			tests: []string{
				"internal/term/defaults/register_test.go",
			},
		},
		{
			item: "Full regex search command tests.",
			tests: []string{
				"internal/term/defaults/module-search_test.go",
			},
		},
		{
			item: "Theme parse/style tests; generated tests that all four " +
				"Catppuccin variants parse.",
			tests: []string{
				"internal/term/theme/theme_test.go",
				"internal/loader/theme_test.go",
			},
		},
		{
			item: "Full config parse/merge coverage for the modeled config " +
				"surface.",
			tests: []string{
				"internal/view/config/config_test.go",
				"internal/view/config-types_test.go",
				"internal/term/defaults/module-config_test.go",
			},
		},
		{
			item: "Generated tests: every supported language entry parses; " +
				"every supported runtime query file is discoverable.",
			tests: []string{
				"internal/term/syntax/syntax_test.go",
			},
		},
	}
}

func planItemChecked(plan, item string) bool {
	return strings.Contains(plan, "- [x] "+item)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	assert.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
