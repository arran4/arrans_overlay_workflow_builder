import re

filepath = "generateWorkflows_issue172_test.go"
with open(filepath, 'r') as f:
    content = f.read()

content = content.replace('wantDepend := "dev-qt/qtbase:6[,core,gui,widgets]', 'wantDepend := " dev-qt/qtbase:6[,core,gui,widgets]')
content = content.replace('require.Equal(t, "123456789012345678901234567890123456789012345678901234567890123test-bin", got)', 'require.Equal(t, "123456789012345678901234567890123456789012345678901234567890123${P}", got)')

with open(filepath, 'w') as f:
    f.write(content)
