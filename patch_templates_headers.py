import re

files = [
    "templates/github-binary.tmpl",
    "templates/web-binary.tmpl",
    "templates/github-appimage.tmpl",
    "templates/web-appimage.tmpl",
    "templates/github-cmake.tmpl"
]

for filepath in files:
    with open(filepath, 'r') as f:
        content = f.read()

    # Match the typical Gentoo Authors header block
    content = re.sub(
        r"echo '# Copyright \[\[ .Now.Format \"2006\" \]\] Gentoo Authors'\n\s+echo '# Distributed under the terms of the GNU General Public License v2'",
        r'[[ .GeneratedEbuildHeader | shell_echo | indent (repeat 14 " ") ]]',
        content
    )

    # In github-cmake.tmpl it uses $(date +%Y)
    content = re.sub(
        r"echo \"# Copyright \$\(date \+%Y\) Gentoo Authors\"\n\s+echo \"# Distributed under the terms of the GNU General Public License v2\"",
        r'[[ .GeneratedEbuildHeader | shell_echo | indent (repeat 14 " ") ]]',
        content
    )

    with open(filepath, 'w') as f:
        f.write(content)
