with open("generateWorkflows.go", "r") as f:
    content = f.read()

content = content.replace('if originalVersion == true || originalVersion == "true" {', 'if fmt.Sprintf("%v", originalVersion) == "true" {')

with open("generateWorkflows.go", "w") as f:
    f.write(content)
