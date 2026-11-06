package arrans_overlay_workflow_builder

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/arran4/arrans_overlay_workflow_builder/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failingMkdirFS simulates a filesystem where MkdirAll fails.
type failingMkdirFS struct {
	util.MockFS
	mkdirErr error
}

func (f *failingMkdirFS) MkdirAll(path string, perm fs.FileMode) error {
	return f.mkdirErr
}

// failingWriteNthFS simulates a filesystem where the N-th WriteFile fails.
type failingWriteNthFS struct {
	util.MockFS
	writeCount int
	failOnCall int
	writeErr   error
}

func (f *failingWriteNthFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	f.writeCount++
	if f.writeCount == f.failOnCall {
		return f.writeErr
	}
	return f.MockFS.WriteFile(name, data, perm)
}

func makeValidConfig(category, name string) *InputConfig {
	return &InputConfig{
		Category:   category,
		EbuildName: name,
		Type:       "Github Binary Release",
	}
}

// TestIssue180_Regression_LateRenderError_OverwritesPreexistingFile demonstrates
// the unpatched defect: when batch workflow generation encounters a late configuration
// or template rendering error, earlier workflows have already been written or overwritten.
func TestIssue180_Regression_LateRenderError_OverwritesPreexistingFile(t *testing.T) {
	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(outputDir, 0755))

	cfg1 := makeValidConfig("app-misc", "pkg1-bin")
	cfg2 := makeValidConfig("app-misc", "pkg2-bin")
	cfg2.Type = "Unsupported Invalid Type" // Will fail at render stage

	file1 := filepath.Join(outputDir, "app-misc-pkg1-bin-update.yaml")
	file2 := filepath.Join(outputDir, "app-misc-pkg2-bin-update.yaml")

	originalContent := []byte("# PRE-EXISTING WORKFLOW DATA THAT MUST BE PRESERVED")
	require.NoError(t, os.WriteFile(file1, originalContent, 0644))

	fixedTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	err := GenerateGithubWorkflowsFromInputConfigs("test.config", []*InputConfig{cfg1, cfg2}, outputDir, "v1.0.0", fixedTime)
	require.Error(t, err, "batch generation must fail due to cfg2 invalid type")

	// In the fixed implementation, file1 MUST NOT be overwritten with new generated content,
	// and file2 must not exist.
	currentContent, readErr := os.ReadFile(file1)
	require.NoError(t, readErr)
	assert.Equal(t, string(originalContent), string(currentContent),
		"nonmutation contract: pre-existing file1 content must be preserved when late configuration fails")

	_, statErr := os.Stat(file2)
	assert.True(t, os.IsNotExist(statErr), "file2 must not exist after failed generation")
}

// TestIssue180_Regression_LateRenderError_InitiallyAbsentFileNotCreated asserts that
// no partial workflow files are created when a late configuration fails.
func TestIssue180_Regression_LateRenderError_InitiallyAbsentFileNotCreated(t *testing.T) {
	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, ".github", "workflows")

	cfg1 := makeValidConfig("app-misc", "first-bin")
	cfg2 := makeValidConfig("app-misc", "second-bin")
	cfg2.Type = "Unknown Type" // Will fail at render stage

	file1 := filepath.Join(outputDir, "app-misc-first-bin-update.yaml")
	file2 := filepath.Join(outputDir, "app-misc-second-bin-update.yaml")

	fixedTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	err := GenerateGithubWorkflowsFromInputConfigs("test.config", []*InputConfig{cfg1, cfg2}, outputDir, "v1.0.0", fixedTime)
	require.Error(t, err)

	_, err1 := os.Stat(file1)
	assert.True(t, os.IsNotExist(err1), "file1 must not be left behind when later entry fails")
	_, err2 := os.Stat(file2)
	assert.True(t, os.IsNotExist(err2), "file2 must not be left behind when later entry fails")
}

// TestIssue180_MkdirAllError_Propagated asserts that errors during directory creation
// are propagated with actionable context rather than silently swallowed.
func TestIssue180_MkdirAllError_Propagated(t *testing.T) {
	mockFS := &failingMkdirFS{
		MockFS:   *util.NewMockFS(),
		mkdirErr: errors.New("permission denied creating workflows directory"),
	}

	cfg := makeValidConfig("app-misc", "pkg-bin")
	err := GenerateGithubWorkflowsFromInputConfigs("test.config", []*InputConfig{cfg}, "/nonexistent/workflows", "v1.0.0", mockFS)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "creating output directory")
	assert.True(t, errors.Is(err, mockFS.mkdirErr) || errors.Unwrap(err) != nil, "original mkdir error must be wrapped")
}

// TestIssue180_PublicationWriteFailure_RollbackRestoresState asserts that if an I/O
// error occurs during the publication write loop, previously written files in the batch
// are rolled back (pre-existing restored, newly created removed).
func TestIssue180_PublicationWriteFailure_RollbackRestoresState(t *testing.T) {
	mockFS := &failingWriteNthFS{
		MockFS:     *util.NewMockFS(),
		failOnCall: 2, // Second write will fail
		writeErr:   errors.New("disk full writing second file"),
	}

	path1 := "/workflows/app-misc-pkg1-bin-update.yaml"
	path2 := "/workflows/app-misc-pkg2-bin-update.yaml"

	// Pre-populate path1 with existing content
	orig1 := []byte("ORIGINAL_PKG1_CONTENT")
	require.NoError(t, mockFS.WriteFile(path1, orig1, 0644))

	// Reset writeCount so failOnCall applies during batch generation
	mockFS.writeCount = 0

	cfg1 := makeValidConfig("app-misc", "pkg1-bin")
	cfg2 := makeValidConfig("app-misc", "pkg2-bin")

	err := GenerateGithubWorkflowsFromInputConfigs("test.config", []*InputConfig{cfg1, cfg2}, "/workflows", "v1.0.0", mockFS)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disk full writing second file")

	// Assert path1 content was restored to orig1
	content1, readErr1 := mockFS.ReadFile(path1)
	require.NoError(t, readErr1)
	assert.Equal(t, string(orig1), string(content1), "pre-existing file1 must be restored after write failure")

	// Assert path2 does not exist
	_, readErr2 := mockFS.ReadFile(path2)
	assert.True(t, errors.Is(readErr2, os.ErrNotExist), "path2 must not exist in filesystem after write failure")
}

// TestIssue180_SuccessfulBatchGeneration_PreservesAllOutputs asserts that normal
// multi-file batch generation succeeds cleanly and produces all expected workflow files.
func TestIssue180_SuccessfulBatchGeneration_PreservesAllOutputs(t *testing.T) {
	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, ".github", "workflows")

	cfg1 := makeValidConfig("app-misc", "tool1-bin")
	cfg2 := makeValidConfig("app-misc", "tool2-bin")

	fixedTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	err := GenerateGithubWorkflowsFromInputConfigs("test.config", []*InputConfig{cfg1, cfg2}, outputDir, "v1.0.0", fixedTime)
	require.NoError(t, err)

	file1 := filepath.Join(outputDir, "app-misc-tool1-bin-update.yaml")
	file2 := filepath.Join(outputDir, "app-misc-tool2-bin-update.yaml")

	bytes1, err1 := os.ReadFile(file1)
	require.NoError(t, err1)
	assert.Contains(t, string(bytes1), "name: app-misc/tool1-bin update")

	bytes2, err2 := os.ReadFile(file2)
	require.NoError(t, err2)
	assert.Contains(t, string(bytes2), "name: app-misc/tool2-bin update")
}
