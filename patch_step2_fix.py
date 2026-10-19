import re

files = [
    "templates/github-binary.tmpl",
    "templates/github-appimage.tmpl",
    "templates/github-cmake.tmpl",
    "templates/web-binary.tmpl",
    "templates/web-appimage.tmpl"
]

for filepath in files:
    with open(filepath, 'r') as f:
        content = f.read()

    # Need to replace emit_metadata_field "SRC_URI" with emit_ebuild_assignment "SRC_URI"
    content = content.replace('emit_metadata_field "SRC_URI"', 'emit_ebuild_assignment "SRC_URI"')
    # Same for DESCRIPTION, HOMEPAGE, LICENSE, KEYWORDS, IUSE, REQUIRED_USE, RDEPEND, DEPEND
    content = content.replace('emit_metadata_field "DESCRIPTION"', 'emit_ebuild_assignment "DESCRIPTION"')
    content = content.replace('emit_metadata_field "HOMEPAGE"', 'emit_ebuild_assignment "HOMEPAGE"')
    content = content.replace('emit_metadata_field "LICENSE"', 'emit_ebuild_assignment "LICENSE"')
    content = content.replace('emit_metadata_field "KEYWORDS"', 'emit_ebuild_assignment "KEYWORDS"')
    content = content.replace('emit_metadata_field "IUSE"', 'emit_ebuild_assignment "IUSE"')
    content = content.replace('emit_metadata_field "REQUIRED_USE"', 'emit_ebuild_assignment "REQUIRED_USE"')
    content = content.replace('emit_metadata_field "RDEPEND"', 'emit_ebuild_assignment "RDEPEND"')
    content = content.replace('emit_metadata_field "DEPEND"', 'emit_ebuild_assignment "DEPEND"')

    # Inject `[[ template "ebuild_output_helpers" . ]]` before the tag loop
    if filepath == "templates/github-cmake.tmpl":
        content = re.sub(r'(\s+)(tags=\$\(gh api)', r'\1[[ template "ebuild_output_helpers" . ]]\1\2', content)
    else:
        # Before PreVersionSteps, or after declare -A releaseTypes=()
        content = re.sub(r'(\s+declare -A releaseTypes=\(\))', r'\1\n[[ template "ebuild_output_helpers" . ]]', content)

    with open(filepath, 'w') as f:
        f.write(content)
