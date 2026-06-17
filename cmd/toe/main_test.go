package main_test

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	t.Run("health", func(t *testing.T) {
		cmd := exec.Command("go", "run", ".", "--health")
		var out bytes.Buffer
		cmd.Stdout = &out

		err := cmd.Run()

		assert.NoError(t, err)
		assert.Contains(t, out.String(), "toe health: ok")
	})
}
