#!/bin/bash
find templates/ -name "*.tmpl" -exec sed -i -E 's/echo "\$\{\{ github.repository \}\}" > profiles\/repo_name/echo "\$\{GITHUB_REPOSITORY#*\/\}" > profiles\/repo_name/' {} +
