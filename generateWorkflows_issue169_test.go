package arrans_overlay_workflow_builder

import (
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	// Exercise the actual CLI entrypoint once, without abandoned subprocess setup.
	cmd := exec.Command("go", "run", "./cmd/generate/main.go", "generate", "workflows", "-input-file", inputPath, "-output-dir", tempDir)
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

		gentooAuthors, sourceLicense, hasExplicitSourceLicense := ic.ResolveHeaderPolicy(ops...)
		base := &GenerateGithubWorkflowBase{
			Now:                      time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC),
			InputConfig:              ic,
			GentooAuthors:            gentooAuthors,
			SourceLicense:            sourceLicense,
			HasExplicitSourceLicense: hasExplicitSourceLicense,
		}

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

// TestIssue169HeaderOptionsRoundTrip protects the distinction between an
// unspecified option and an explicitly set option. SourceLicense controls
// the generated source header; License is the ebuild package license.
func TestIssue169HeaderOptionsRoundTrip(t *testing.T) {
	boolPtr := func(v bool) *bool { return &v }
	stringPtr := func(v string) *string { return &v }

	tests := []struct {
		name        string
		directives  string
		wantAuthors *bool
		wantSource  *string
	}{
		{name: "both-unset"},
		{name: "authors-false", directives: "GentooAuthors false\n", wantAuthors: boolPtr(false)},
		{name: "authors-true", directives: "GentooAuthors true\n", wantAuthors: boolPtr(true)},
		{name: "source-MIT-explicit", directives: "SourceLicense MIT\n", wantSource: stringPtr("MIT")},
		{name: "source-Apache", directives: "SourceLicense Apache-2.0\n", wantSource: stringPtr("Apache-2.0")},
		{
			name:        "authors-false-source-Apache",
			directives:  "GentooAuthors false\nSourceLicense Apache-2.0\n",
			wantAuthors: boolPtr(false),
			wantSource:  stringPtr("Apache-2.0"),
		},
		{
			name:        "authors-true-source-MIT",
			directives:  "GentooAuthors true\nSourceLicense MIT\n",
			wantAuthors: boolPtr(true),
			wantSource:  stringPtr("MIT"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			source := "Type Github Binary Release\n" +
				"GithubProjectUrl https://github.com/test/test\n" +
				"Category app-admin\n" +
				"EbuildName test-bin\n" +
				"Description Test\n" +
				"Homepage https://example.test/\n" +
				"License GPL-3.0\n" +
				tc.directives +
				"Binary amd64=>test-\u0024{TAG}.tar.gz > test > test\n"

			initial, err := ParseInputConfigReader(strings.NewReader(source))
			require.NoError(t, err)
			require.Len(t, initial, 1)
			require.Equal(t, tc.wantAuthors, initial[0].GentooAuthors)
			require.Equal(t, tc.wantSource, initial[0].SourceLicense)
			require.Equal(t, "GPL-3.0", initial[0].License)

			serialized := initial[0].String()
			if tc.wantAuthors == nil {
				require.NotContains(t, serialized, "GentooAuthors ")
			}
			if tc.wantSource == nil {
				require.NotContains(t, serialized, "SourceLicense ")
			}

			roundTrip, err := ParseInputConfigReader(strings.NewReader(serialized))
			require.NoError(t, err)
			require.Len(t, roundTrip, 1)
			require.Equal(t, tc.wantAuthors, roundTrip[0].GentooAuthors)
			require.Equal(t, tc.wantSource, roundTrip[0].SourceLicense)
			require.Equal(t, initial[0].License, roundTrip[0].License,
				"the ebuild package license must remain independent of source-header configuration")
		})
	}
}
