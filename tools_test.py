import re
import os

files = [
    "templates/github-appimage.tmpl",
    "templates/web-binary.tmpl",
    "templates/github-binary.tmpl",
    "templates/web-appimage.tmpl",
    "templates/github-cmake.tmpl"
]

for filepath in files:
    with open(filepath, "r") as f:
        content = f.read()

    new_content = re.sub(
        r'          action: \'lint \. \$\{\{ env\.ecn \}\}/\$\{\{ env\.epn \}\}\'',
        r'          action: \'lint [[ if not .FeatureGenerateMd5Cache ]]-disable-rule PG0802 [[ end ]]. ${{ env.ecn }}/${{ env.epn }}\'',
        content,
    )

    with open(filepath, "w") as f:
        f.write(new_content)

print("done")
