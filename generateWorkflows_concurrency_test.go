package arrans_overlay_workflow_builder

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGeneratedWorkflowsHaveConcurrencyGroup verifies that all generated workflow templates
// correctly configure their concurrency groups.
//
// We assert the following exactly:
// 1. group: ci-${{ github.workflow_ref }} - To prevent same-name but different-file workflows from colliding, and to isolate executions across branches/refs.
// 2. cancel-in-progress: false - To ensure we do not cancel pending runs (with a max of 100).
// 3. queue: max - To ensure the jobs are queued properly up to 100 instead of failing.
func TestGeneratedWorkflowsHaveConcurrencyGroup(t *testing.T) {
	templatesToTest := []string{
		"templates/github-binary.tmpl",
		"templates/github-appimage.tmpl",
		"templates/github-cmake.tmpl",
		"templates/web-binary.tmpl",
		"templates/web-appimage.tmpl",
	}

	for _, tmplFile := range templatesToTest {
		t.Run(tmplFile, func(t *testing.T) {
			contentBytes, err := os.ReadFile(tmplFile)
			assert.NoError(t, err)
			content := string(contentBytes)

			assert.Contains(t, content, "concurrency:")
			assert.Contains(t, content, "group: ci-${{ github.workflow_ref }}")
			assert.Contains(t, content, "cancel-in-progress: false")
			assert.Contains(t, content, "queue: max")
		})
	}
}
