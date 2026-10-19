import re

filepath = "generateWorkflows_issue172_test.go"
with open(filepath, 'r') as f:
    content = f.read()

content = content.replace('evalScript := strings.Join(assignments, "\\n") + `\nprintf \'%s\\0%s\\0%s\' "$DEPEND" "$RDEPEND" "$BDEPEND"\n`',
'evalScript := strings.Join(assignments, "\\n") + `\nprintf \'%s\\0%s\\0%s\' "$DEPEND" "${RDEPEND:-$DEPEND}" "$BDEPEND"\n`')

with open(filepath, 'w') as f:
    f.write(content)
