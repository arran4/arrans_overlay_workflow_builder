import re

with open("templates/_partials/src_uri.tmpl", "r") as f:
    content = f.read()

content = re.sub(
    r'\|\s*ebuildvardoublequoted[^\s|]*\s*\|\s*replace\s*"\\+\$\{PV\}"\s*"\$\{originalVersion\}"',
    r'| formatReleaseFilenameForUrl $.WorkaroundSemanticVersionPrereleaseHack1',
    content
)

with open("templates/_partials/src_uri.tmpl", "w") as f:
    f.write(content)

with open("templates/_partials/manifest_upsert.tmpl", "r") as f:
    content = f.read()

content = re.sub(
    r'\|\s*actionvardoublequoted\s*\|\s*replace\s*"\$\{version\}"\s*"\$\{originalVersion\}"',
    r'| formatReleaseFilenameForUrl $.WorkaroundSemanticVersionPrereleaseHack1',
    content
)

content = re.sub(
    r'\|\s*ebuildvardoublequoted[^\s|]*\s*\|\s*replace\s*"\\+\$\{PV\}"\s*"\$\{originalVersion\}"\s*\|\s*replace\s*"\\+\$\{P\}"\s*"\$\{\{\s*env\.epn\s*\}\}-\$\{originalVersion\}"',
    r'| formatReleaseFilenameForUrl $.WorkaroundSemanticVersionPrereleaseHack1',
    content
)
content = re.sub(
    r'\|\s*actionvardoublequoted\s*\|\s*replace\s*"\\+\$\{PV\}"\s*"\$\{originalVersion\}"\s*\|\s*replace\s*"\\+\$\{P\}"\s*"\$\{\{\s*env\.epn\s*\}\}-\$\{originalVersion\}"',
    r'| formatReleaseFilenameForUrl $.WorkaroundSemanticVersionPrereleaseHack1',
    content
)

with open("templates/_partials/manifest_upsert.tmpl", "w") as f:
    f.write(content)

with open("templates/_partials/ebuild_revision_and_manifest.tmpl", "r") as f:
    content = f.read()

content = re.sub(
    r'\|\s*ebuildvardoublequoted[^\s|]*\s*\|\s*replace\s*"\\+\$\{PV\}"\s*"\$\{originalVersion\}"',
    r'| formatReleaseFilenameForUrl $.WorkaroundSemanticVersionPrereleaseHack1',
    content
)
content = re.sub(
    r'\|\s*actionvardoublequoted\s*\|\s*replace\s*"\\+\$\{PV\}"\s*"\$\{originalVersion\}"',
    r'| formatReleaseFilenameForUrl $.WorkaroundSemanticVersionPrereleaseHack1',
    content
)
content = re.sub(
    r'\|\s*actionvardoublequoted\s*\|\s*replace\s*"\$\{version\}"\s*"\$\{originalVersion\}"',
    r'| formatReleaseFilenameForUrl $.WorkaroundSemanticVersionPrereleaseHack1',
    content
)


with open("templates/_partials/ebuild_revision_and_manifest.tmpl", "w") as f:
    f.write(content)
