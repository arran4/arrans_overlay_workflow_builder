import re

with open("smoke_test.go", "r") as f:
    content = f.read()

content = re.sub(r't\.Logf\("Evaluated SRC_URI:\\n%s", evaluatedSRC_URI\)\n\s*t\.Logf\("Script:\\n%s", srcURIScript\)', '', content)

with open("smoke_test.go", "w") as f:
    f.write(content)
