import re

filepath = "generateWorkflows_issue172_test.go"
with open(filepath, 'r') as f:
    content = f.read()

content = content.replace('script = strings.ReplaceAll(script, "emit_metadata_field \\"DESCRIPTION\\" \\"Test\\"", "emit_metadata_field \\"DESCRIPTION\\" \\"123456789012345678901234567890123456789012345678901234567890123\\\\${P}\\"")',
'script = strings.ReplaceAll(script, "emit_ebuild_assignment \\"DESCRIPTION\\" \\"Test\\"", "emit_ebuild_assignment \\"DESCRIPTION\\" \\"123456789012345678901234567890123456789012345678901234567890123\\\\${P}\\"")')

content = content.replace('wantDepend := " dev-qt/qtbase:6[,core,gui,widgets]', 'wantDepend := "dev-qt/qtbase:6[,core,gui,widgets]')

with open(filepath, 'w') as f:
    f.write(content)
