import re

filepath = "generateWorkflows_test.go"
with open(filepath, 'r') as f:
    content = f.read()

content = content.replace('emit_metadata_field \\"SRC_URI\\"', 'emit_ebuild_assignment \\"SRC_URI\\"')

with open(filepath, 'w') as f:
    f.write(content)

import glob
for file in glob.glob("templates/*.tmpl"):
    with open(file, 'r') as f:
        content = f.read()
    content = content.replace('emit_metadata_field "SRC_URI"', 'emit_ebuild_assignment "SRC_URI"')
    with open(file, 'w') as f:
        f.write(content)
