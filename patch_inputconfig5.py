import re

gw = "generateWorkflows.go"
with open(gw, 'r') as f:
    content = f.read()

content = content.replace("GlobalUpsertOverlayRepoName bool\n", "GlobalUpsertOverlayRepoName bool\n\tGlobalGeneratedEbuildGentooAuthors *bool\n\tGlobalGeneratedEbuildSourceLicense *string\n")

with open(gw, 'w') as f:
    f.write(content)
