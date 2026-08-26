package arrans_overlay_workflow_builder

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestIssue126ConcurrencyIndentation(t *testing.T) {
	// A minimal configuration for Github Binary Release
	config := `Type Github Binary Release
GithubProjectUrl https://github.com/test/test
EbuildName test-bin
Category app-admin
Description Test
Homepage https://www.test.io/
License MIT License
`
	inputConfigs, err := ParseInputConfigReader(strings.NewReader(config))
	assert.NoError(t, err)
	assert.Len(t, inputConfigs, 1)

	templates, err := ParseWorkflowTemplates()
	assert.NoError(t, err)

	ic := inputConfigs[0]
	now := time.Now()

	// Create data to render template manually
	base := &GenerateGithubWorkflowBase{
		Version:     "1.0",
		InputConfig: ic,
		ConfigFile:  "test.config",
		Now:         now.UTC(),
	}
	data := &GenerateGithubBinaryTemplateData{
		GenerateGithubWorkflowBase: base,
	}

	var out bytes.Buffer
	err = templates.ExecuteTemplate(&out, "github-binary.tmpl", data)
	assert.NoError(t, err)

	w := out.Bytes()

	// Structurally verify parsing as YAML
	var m map[string]interface{}
	err = yaml.Unmarshal(w, &m)
	assert.NoError(t, err, "YAML parsing failed, indicating mapping issues such as incorrect concurrency indentation")

	_, hasEnv := m["env"]
	assert.True(t, hasEnv, "env mapping should exist at root")

	_, hasConcurrency := m["concurrency"]
	assert.True(t, hasConcurrency, "concurrency mapping should exist at root")

	_, hasJobs := m["jobs"]
	assert.True(t, hasJobs, "jobs mapping should exist at root")

	// Check that concurrency isn't nested under env
	envMap, ok := m["env"].(map[string]interface{})
	if ok {
		_, envHasConcurrency := envMap["concurrency"]
		assert.False(t, envHasConcurrency, "concurrency should NOT be nested inside env")
	}

	// Verify exact structure in text (concurrency at root)
	assert.Contains(t, string(w), "\nconcurrency:\n")
}
