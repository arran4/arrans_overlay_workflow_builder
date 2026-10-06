#!/bin/bash
set -e

# 1. inputconfig.go
cat << 'PY' > fix_inputconfig.py
import re

with open("inputconfig.go", "r") as f:
    content = f.read()

content = re.sub(
    r'(type InputConfig struct \{)',
    r'\1\n\tGentooAuthors       *bool\n\tSourceLicense       *string',
    content,
    count=1
)

content = re.sub(
    r'(\t\tif ic\.License != "" \{\n\t\t\tfmt\.Fprintf\(&sb, "License %s\\n", ic\.License\)\n\t\t\})',
    r'\1\n\t\tif ic.GentooAuthors != nil {\n\t\t\tfmt.Fprintf(&sb, "GentooAuthors %v\\n", *ic.GentooAuthors)\n\t\t}\n\t\tif ic.SourceLicense != nil {\n\t\t\tfmt.Fprintf(&sb, "SourceLicense %s\\n", *ic.SourceLicense)\n\t\t}',
    content
)

original_parse = """	currentConfig.License, err = emptyOrLast(parsedFields["License"])
	if err != nil {
		return nil, fmt.Errorf("on License: %v: %w", parsedFields["License"], err)
	}"""
replacement_parse = """	currentConfig.License, err = emptyOrLast(parsedFields["License"])
	if err != nil {
		return nil, fmt.Errorf("on License: %v: %w", parsedFields["License"], err)
	}
	currentConfig.GentooAuthors = emptyOrLastBoolPtr(parsedFields["GentooAuthors"])
	currentConfig.SourceLicense = emptyOrLastPtr(parsedFields["SourceLicense"])"""

content = content.replace(original_parse, replacement_parse)

helpers = """
func emptyOrLastBoolPtr(i []string) *bool {
	if len(i) == 0 {
		return nil
	}
	v := strings.ToLower(strings.TrimSpace(i[len(i)-1]))
	b := v == "true" || v == "yes" || v == "1" || v == "on"
	return &b
}

func emptyOrLastPtr(i []string) *string {
	if len(i) == 0 {
		return nil
	}
	v := i[len(i)-1]
	return &v
}
"""

if "emptyOrLastBoolPtr" not in content:
    content = content + helpers

with open("inputconfig.go", "w") as f:
    f.write(content)
PY
python3 fix_inputconfig.py

# 2. cmd/generate/main.go
cat << 'PY' > fix_generate_main.py
import re

with open("cmd/generate/main.go", "r") as f:
    content = f.read()

content = content.replace(
    'upsertOverlayRepoName := generateWorkflowsCmd.Bool("upsert-overlay-repo-name", false, "Creates profiles/repo_name if missing, or validates it if present.")',
    'upsertOverlayRepoName := generateWorkflowsCmd.Bool("upsert-overlay-repo-name", false, "Creates profiles/repo_name if missing, or validates it if present.")\n\tgentooAuthors := generateWorkflowsCmd.Bool("gentoo-authors", false, "Emit Gentoo Authors copyright header instead of generic MIT.")\n\tsourceLicense := generateWorkflowsCmd.String("source-license", "MIT", "Source-level license to emit in headers (e.g., MIT).")'
)

content = content.replace(
    'aowb.OptUpsertOverlayRepoName(*upsertOverlayRepoName),',
    'aowb.OptUpsertOverlayRepoName(*upsertOverlayRepoName),\n\t\taowb.OptGentooAuthors(*gentooAuthors),\n\t\taowb.OptSourceLicense(*sourceLicense),'
)

with open("cmd/generate/main.go", "w") as f:
    f.write(content)
PY
python3 fix_generate_main.py

# 3. generateWorkflows.go
cat << 'PY' > fix_gen_options.py
import re

with open("generateWorkflows.go", "r") as f:
    content = f.read()

content = content.replace(
    'type OptUpsertOverlayRepoName bool',
    'type OptUpsertOverlayRepoName bool\ntype OptGentooAuthors bool\ntype OptSourceLicense string'
)

content = content.replace(
    'GlobalUpsertOverlayRepoName bool',
    'GlobalUpsertOverlayRepoName bool\n\tGentooAuthors               bool\n\tSourceLicense               string'
)

original_base = """	base := &GenerateGithubWorkflowBase{
		Version:     version,
		Now:         now,
		ConfigFile:  file,
		InputConfig: ic,
	}"""
new_base = """	gentooAuthors := false
	sourceLicense := "MIT"

	for _, opt := range ops {
		switch o := opt.(type) {
		case OptGentooAuthors:
			gentooAuthors = bool(o)
		case OptSourceLicense:
			sourceLicense = string(o)
		}
	}

	if ic.GentooAuthors != nil {
		gentooAuthors = *ic.GentooAuthors
	}
	if ic.SourceLicense != nil && *ic.SourceLicense != "" {
		sourceLicense = *ic.SourceLicense
	}

	base := &GenerateGithubWorkflowBase{
		Version:       version,
		Now:           now,
		ConfigFile:    file,
		InputConfig:   ic,
		GentooAuthors: gentooAuthors,
		SourceLicense: sourceLicense,
	}"""

content = content.replace(original_base, new_base)

with open("generateWorkflows.go", "w") as f:
    f.write(content)
PY
python3 fix_gen_options.py

# 4. generateGithubCmakeWorkflow.go
cat << 'PY' > fix_cmake.py
with open("generateGithubCmakeWorkflow.go", "r") as f:
    content = f.read()

original_struct = """type GenerateGithubCmakeTemplateData struct {
	*GenerateGithubWorkflowBase
}"""
new_struct = """type GenerateGithubCmakeTemplateData struct {
	*GenerateGithubWorkflowBase
	GentooAuthors bool
	SourceLicense string
}"""

original_init = """	return &GenerateGithubCmakeTemplateData{
		GenerateGithubWorkflowBase: base,
	}"""
new_init = """	return &GenerateGithubCmakeTemplateData{
		GenerateGithubWorkflowBase: base,
		GentooAuthors: base.GentooAuthors,
		SourceLicense: base.SourceLicense,
	}"""

content = content.replace(original_struct, new_struct)
content = content.replace(original_init, new_init)

with open("generateGithubCmakeWorkflow.go", "w") as f:
    f.write(content)
PY
python3 fix_cmake.py

# 5. templates
cat << 'PY' > fix_templates.py
import os

header_logic = """[[- if .GentooAuthors ]]
                echo '# Copyright [[ .Now.Format "2006" ]] Gentoo Authors'
                echo '# Distributed under the terms of the GNU General Public License v2'
[[- else ]]
                echo "# SPDX-License-Identifier: [[ .SourceLicense ]]"
[[- end ]]"""

header_logic2 = """[[- if .GentooAuthors ]]
              echo '# Copyright [[ .Now.Format "2006" ]] Gentoo Authors'
              echo '# Distributed under the terms of the GNU General Public License v2'
[[- else ]]
              echo "# SPDX-License-Identifier: [[ .SourceLicense ]]"
[[- end ]]"""

for filename in os.listdir("templates"):
    if not filename.endswith(".tmpl"): continue

    with open(f"templates/{filename}", "r") as f:
        content = f.read()

    original_copyright = """                echo '# Copyright [[ .Now.Format "2006" ]] Gentoo Authors'
                echo '# Distributed under the terms of the GNU General Public License v2'"""

    content = content.replace(original_copyright, header_logic)

    original_copyright2 = """              echo '# Copyright [[ .Now.Format "2006" ]] Gentoo Authors'
              echo '# Distributed under the terms of the GNU General Public License v2'"""
    content = content.replace(original_copyright2, header_logic2)

    with open(f"templates/{filename}", "w") as f:
        f.write(content)
PY
python3 fix_templates.py

# 6. smoke_test.go
cat << 'PY' > fix_smoke.py
with open("smoke_test.go", "r") as f:
    content = f.read()

original = """					// Assert 1: Standard header
					require.Contains(t, string(ebuildContent), "# Copyright 2026 Gentoo Authors", "Ebuild missing Copyright header")
					require.Contains(t, string(ebuildContent), "# Distributed under the terms of the GNU General Public License v2", "Ebuild missing License header")"""

replacement = """					// Assert 1: Standard header
					require.Contains(t, string(ebuildContent), "# SPDX-License-Identifier:", "Ebuild missing SPDX header")"""

content = content.replace(original, replacement)

with open("smoke_test.go", "w") as f:
    f.write(content)
PY
python3 fix_smoke.py

rm *.py
