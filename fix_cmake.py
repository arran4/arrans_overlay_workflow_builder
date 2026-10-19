import re

filepath = "templates/github-cmake.tmpl"
with open(filepath, 'r') as f:
    content = f.read()

content = content.replace('emit_metadata_field "BDEPEND"', 'emit_ebuild_assignment "BDEPEND"')

with open(filepath, 'w') as f:
    f.write(content)
