package defaults_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
)

type vcsStub struct {
	hunks []view.DiffHunk
	base  string
}

var _ view.VersionControl = (*vcsStub)(nil)

func (s *vcsStub) DiffHunks(*view.Document) []view.DiffHunk { return s.hunks }
func (s *vcsStub) DiffHunksForPath(string) []view.DiffHunk  { return s.hunks }
func (s *vcsStub) HeadName(*view.Document) (string, bool)   { return "", false }
func (s *vcsStub) ChangedFiles() ([]view.FileChange, error) { return nil, nil }
func (s *vcsStub) Refresh()                                 {}
func (s *vcsStub) Updates() <-chan struct{}                 { return nil }
func (s *vcsStub) DiffBase(*view.Document) (string, bool) {
	return s.base, s.base != ""
}

func TestVCSModule(t *testing.T) {
	t.Run("reset diff reports error without vc", func(t *testing.T) {
		e, km := defaultsEnv(t, "one\ntwo\n")
		res := runCmd(t, km, e, "reset_diff_change")
		assert.True(t, strings.HasPrefix(res.Message, "error:"))
	})

	t.Run("reset diff resets one change", func(t *testing.T) {
		e, km := defaultsEnv(t, "one\nCHANGED\nthree\n")
		e.SetVersionControl(&vcsStub{
			base: "one\ntwo\nthree\n",
			hunks: []view.DiffHunk{
				{BaseFrom: 1, BaseTo: 2, From: 1, To: 2},
			},
		})
		testutil.SetCursor(t, e, 4) // inside "CHANGED" line
		res := runCmd(t, km, e, "reset_diff_change")
		assert.Equal(t, "Reset 1 change", res.Message)
	})

	t.Run("reset diff resets multiple changes", func(t *testing.T) {
		e, km := defaultsEnv(t, "one\nCHANGED\nthree\nADDED\n")
		e.SetVersionControl(&vcsStub{
			base: "one\ntwo\nthree\n",
			hunks: []view.DiffHunk{
				{BaseFrom: 1, BaseTo: 2, From: 1, To: 2},
				{BaseFrom: 3, BaseTo: 3, From: 3, To: 4},
			},
		})
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		// select all lines to cover both hunks
		testutil.SetSelection(t, e, []core.Range{
			core.NewRange(0, doc.Text().LenChars()),
		}, 0)
		res := runCmd(t, km, e, "reset_diff_change")
		assert.Equal(t, "Reset 2 changes", res.Message)
	})
}
