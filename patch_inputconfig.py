import re

inputconfig = "inputconfig.go"
with open(inputconfig, 'r') as f:
    content = f.read()

# Add fields to InputConfig struct
content = content.replace("type InputConfig struct {\n", "type InputConfig struct {\n\tGeneratedEbuildGentooAuthors *bool\n\tGeneratedEbuildSourceLicense *string\n")

# Add parse fields mapping
content = content.replace('"MaintainerName":        nil,', '"MaintainerName":        nil,\n\t\t\t\t"GeneratedEbuildGentooAuthors": nil,\n\t\t\t\t"GeneratedEbuildSourceLicense": nil,')

# Add values loading in CreateSanitizeAndAppendInputConfig
parsing_logic = '''	if val, ok := parsedFields["GeneratedEbuildGentooAuthors"]; ok && len(val) > 0 {
		b := strings.ToLower(val[len(val)-1]) == "true"
		currentConfig.GeneratedEbuildGentooAuthors = &b
	}
	if val, ok := parsedFields["GeneratedEbuildSourceLicense"]; ok && len(val) > 0 {
		s := val[len(val)-1]
		currentConfig.GeneratedEbuildSourceLicense = &s
	}'''
content = content.replace('currentConfig.Programs = parsedProgramFields', parsing_logic + '\n\tcurrentConfig.Programs = parsedProgramFields')

# Add formatting in String()
string_logic = '''
	if ic.GeneratedEbuildGentooAuthors != nil {
		fmt.Fprintf(&sb, "GeneratedEbuildGentooAuthors %t\\n", *ic.GeneratedEbuildGentooAuthors)
	}
	if ic.GeneratedEbuildSourceLicense != nil {
		fmt.Fprintf(&sb, "GeneratedEbuildSourceLicense %s\\n", *ic.GeneratedEbuildSourceLicense)
	}'''
content = content.replace('fmt.Fprintf(&sb, "MaintainerEmail %s\\n", ic.MaintainerEmail)\n\t}', 'fmt.Fprintf(&sb, "MaintainerEmail %s\\n", ic.MaintainerEmail)\n\t}' + string_logic)

with open(inputconfig, 'w') as f:
    f.write(content)

# generateWorkflows.go
gw = "generateWorkflows.go"
with open(gw, 'r') as f:
    content = f.read()

content = content.replace("type OptGenerateMd5Cache bool", "type OptGeneratedEbuildGentooAuthors bool\ntype OptGeneratedEbuildSourceLicense string\ntype OptGenerateMd5Cache bool")

content = content.replace("GlobalUpsertOverlayRepoName bool\n}", "GlobalUpsertOverlayRepoName bool\n\tGlobalGeneratedEbuildGentooAuthors *bool\n\tGlobalGeneratedEbuildSourceLicense *string\n}")

opt_logic = '''		case OptGeneratedEbuildGentooAuthors:
			b := bool(o)
			base.GlobalGeneratedEbuildGentooAuthors = &b
		case OptGeneratedEbuildSourceLicense:
			s := string(o)
			base.GlobalGeneratedEbuildSourceLicense = &s
		case OptGenerateMd5Cache:'''
content = content.replace("case OptGenerateMd5Cache:", opt_logic)

header_logic = '''
func (b *GenerateGithubWorkflowBase) GeneratedEbuildHeader() string {
	gentooAuthors := false
	if b.GlobalGeneratedEbuildGentooAuthors != nil {
		gentooAuthors = *b.GlobalGeneratedEbuildGentooAuthors
	}
	if b.InputConfig.GeneratedEbuildGentooAuthors != nil {
		gentooAuthors = *b.InputConfig.GeneratedEbuildGentooAuthors
	}

	var sourceLicense *string
	if b.GlobalGeneratedEbuildSourceLicense != nil {
		sourceLicense = b.GlobalGeneratedEbuildSourceLicense
	}
	if b.InputConfig.GeneratedEbuildSourceLicense != nil {
		sourceLicense = b.InputConfig.GeneratedEbuildSourceLicense
	}

	var out string
	if gentooAuthors {
		out += fmt.Sprintf("# Copyright %s Gentoo Authors\\n", b.Now.Format("2006"))
		if sourceLicense != nil {
			out += fmt.Sprintf("# SPDX-License-Identifier: %s", *sourceLicense)
		} else {
			out += "# Distributed under the terms of the GNU General Public License v2"
		}
	} else {
		if sourceLicense != nil {
			out += fmt.Sprintf("# SPDX-License-Identifier: %s", *sourceLicense)
		} else {
			out += "# SPDX-License-Identifier: MIT"
		}
	}
	return out
}
'''
content = content.replace("func (b *GenerateGithubWorkflowBase) GetOverlayRepoName() string {", header_logic + "\nfunc (b *GenerateGithubWorkflowBase) GetOverlayRepoName() string {")

with open(gw, 'w') as f:
    f.write(content)

# cmd/generate/main.go
main_go = "cmd/generate/main.go"
with open(main_go, 'r') as f:
    content = f.read()

content = content.replace("UpsertOverlayRepoName bool\n}", "UpsertOverlayRepoName bool\n\tGeneratedEbuildGentooAuthors *bool\n\tGeneratedEbuildSourceLicense *string\n}")

vars_decl = '''
	var globalGeneratedEbuildGentooAuthors *bool
	var globalGeneratedEbuildSourceLicense *string
'''
content = content.replace("var globalUpsertOverlayRepoName bool", "var globalUpsertOverlayRepoName bool\n" + vars_decl)

cases = '''
		case "--generated-ebuild-gentoo-authors", "-generated-ebuild-gentoo-authors":
			if i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "-") {
				b := strings.ToLower(os.Args[i+1]) == "true"
				globalGeneratedEbuildGentooAuthors = &b
				i++
			} else {
				log.Fatalf("missing required value for flag %s", arg)
			}
		case "--generated-ebuild-source-license", "-generated-ebuild-source-license":
			if i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "-") {
				globalGeneratedEbuildSourceLicense = &os.Args[i+1]
				i++
			} else {
				log.Fatalf("missing required value for flag %s", arg)
			}
'''
content = re.sub(r'(\t+)(default:\n\s+newArgs = append\(newArgs, arg\)\n\s+\}\n\s+\})', r'\1' + cases.strip() + r'\n\1\2', content, count=1)

assignment = '''
	config.GeneratedEbuildGentooAuthors = globalGeneratedEbuildGentooAuthors
	config.GeneratedEbuildSourceLicense = globalGeneratedEbuildSourceLicense
'''
content = content.replace("config.OverlayRepoName = &globalOverlayRepoName\n\t}", "config.OverlayRepoName = &globalOverlayRepoName\n\t}\n" + assignment)

opts_append = '''
			if config.GeneratedEbuildGentooAuthors != nil {
				opts = append(opts, arrans_overlay_workflow_builder.OptGeneratedEbuildGentooAuthors(*config.GeneratedEbuildGentooAuthors))
			}
			if config.GeneratedEbuildSourceLicense != nil {
				opts = append(opts, arrans_overlay_workflow_builder.OptGeneratedEbuildSourceLicense(*config.GeneratedEbuildSourceLicense))
			}
'''
content = content.replace("opts = append(opts, arrans_overlay_workflow_builder.OptUpsertOverlayRepoName(true))\n\t\t\t}", "opts = append(opts, arrans_overlay_workflow_builder.OptUpsertOverlayRepoName(true))\n\t\t\t}\n" + opts_append)

with open(main_go, 'w') as f:
    f.write(content)
