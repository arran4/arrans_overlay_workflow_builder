package arrans_overlay_workflow_builder

import (
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestIssue169CLIDefaults(t *testing.T) {
	tempDir, cleanup := setupHermeticEnvironment(t)
	defer cleanup()

	configString := `Type Github Binary Release
GithubProjectUrl https://github.com/test/test
EbuildName test-bin
Category app-admin
Description Test
Homepage https://www.test.io/
License MIT License
Binary amd64=>test-${TAG}-linux-amd64-very-long-asset-name > test > test
Binary arm64=>test-${TAG}-linux-arm64-very-long-asset-name > test > test
`
	inputPath := filepath.Join(tempDir, "test.config")
	err := os.WriteFile(inputPath, []byte(configString), 0644)
	require.NoError(t, err)

	// Wait, testing CLI directly from main package is hard because main is in cmd/generate.
	// I will call `GenerateGithubWorkflows` just like `main.go` does.
	// Or I can build main.go and execute it!

	cmd := exec.Command("go", "run", "../../cmd/generate/main.go", "generate", "workflows", "-input-file", inputPath, "-output-dir", tempDir)
	cmd.Dir = "testdata" // Any dir where go mod is accessible, wait!
	// Since we are running go test from package arrans_overlay_workflow_builder, we can run "go run ./cmd/generate/main.go"
	cmd = exec.Command("go", "run", "./cmd/generate/main.go", "generate", "workflows", "-input-file", inputPath, "-output-dir", tempDir)
	// This executes the CLI path
	err = cmd.Run()
	require.NoError(t, err)

	yamlPath := filepath.Join(tempDir, "app-admin-test-bin-update.yaml")
	yamlData, err := os.ReadFile(yamlPath)
	require.NoError(t, err)

	require.Contains(t, string(yamlData), "# SPDX-License-Identifier: MIT", "CLI default should emit SPDX MIT")
}

func TestIssue169GetHeaderMatrix(t *testing.T) {
	// global false + entry unset -> SPDX MIT
	// global true + entry unset -> Gentoo + GPL-2
	// global false + entry true -> Gentoo + GPL-2
	// global true + entry false -> SPDX MIT
	// global explicit SourceLicense
	// entry SourceLicense overriding global
	// authors=false + explicit Apache-2.0 -> SPDX Apache-2.0
	// authors=true + explicit Apache-2.0 -> Gentoo copyright + SPDX Apache-2.0, no GPL-2 line
	// explicit MIT must remain distinguishable from unset MIT

	// Test helper to generate and check header
	checkHeader := func(globalAuthors bool, globalLicense string, entryAuthors *bool, entryLicense *string, expected string) {
		ic := &InputConfig{
			GentooAuthors: entryAuthors,
			SourceLicense: entryLicense,
		}

		ops := []any{}
		ops = append(ops, OptGentooAuthors(globalAuthors))
		if globalLicense != "" {
			ops = append(ops, OptSourceLicense(globalLicense))
		}

		base := &GenerateGithubWorkflowBase{
			Now:         time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC),
			InputConfig: ic,
		}

		base.ResolveHeaderPolicy(ops...)

		actual := base.GetHeader()
		require.Equal(t, expected, actual)
	}

	tPtr := func(b bool) *bool { return &b }
	sPtr := func(s string) *string { return &s }

	checkHeader(false, "", nil, nil, "# SPDX-License-Identifier: MIT")
	checkHeader(true, "", nil, nil, "# Copyright 2026 Gentoo Authors\n# Distributed under the terms of the GNU General Public License v2")
	checkHeader(false, "", tPtr(true), nil, "# Copyright 2026 Gentoo Authors\n# Distributed under the terms of the GNU General Public License v2")
	checkHeader(true, "", tPtr(false), nil, "# SPDX-License-Identifier: MIT")
	checkHeader(false, "Apache-2.0", nil, nil, "# SPDX-License-Identifier: Apache-2.0")
	checkHeader(false, "Apache-2.0", nil, sPtr("GPL-3.0"), "# SPDX-License-Identifier: GPL-3.0")
	checkHeader(false, "", nil, sPtr("Apache-2.0"), "# SPDX-License-Identifier: Apache-2.0")
	checkHeader(true, "", nil, sPtr("Apache-2.0"), "# Copyright 2026 Gentoo Authors\n# SPDX-License-Identifier: Apache-2.0")
	checkHeader(true, "", nil, sPtr("MIT"), "# Copyright 2026 Gentoo Authors\n# SPDX-License-Identifier: MIT")
}
