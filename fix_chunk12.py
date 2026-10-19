import re

filepath = "templates/_partials/ebuild_output_helpers.tmpl"

# Looking at user's suggestion from trace:
# The user actually gave us a working width-safe `emit_provenance_field` and a skeleton for `emit_ebuild_assignment`.
# Oh! In the prompt, the user said:
# "Do not solve width by placing embedded newlines/tabs in metadata.
# The resulting evaluated value must contain neither: physical newline, physical tab, literal \n, literal \t
# A valid pattern is conceptually:
# DESCRIPTION=""
# DESCRIPTION+=...
# DESCRIPTION+=...
# Use proper shell escaping for each emitted fragment.
# Choose chunk boundaries based on the actual rendered assignment-line width, including: variable name, \+=, shell escaping expansion.
# Do not use a fixed "raw 80 characters" assumption.
# Reject or deliberately handle CR/LF/tab input rather than silently embedding it."

content = """[[ define "ebuild_output_helpers" ]]
          emit_provenance_field() {
              local label=$1
              local value=$2
              local prefix="#   ${label}: "
              local continuation
              local width

              printf -v continuation '%*s' "${#prefix}" ''
              continuation="#${continuation:1}"

              while :; do
                  width=$((80 - ${#prefix}))
                  if ((${#value} <= width)); then
                      printf '%s%s\\n' "$prefix" "$value"
                      break
                  fi
                  printf '%s%s\\n' "$prefix" "${value:0:width}"
                  value="${value:width}"
                  prefix=$continuation
              done
          }

          emit_ebuild_assignment() {
              local name=$1
              local value=$2

              echo "${name}=\\"\\""

              # First clean up whitespace
              local clean_value="${value//$`\\r`/}"
              clean_value="${clean_value//$`\\n`/ }"
              clean_value="${clean_value//$`\\t`/ }"

              local current_chunk=""

              while [ -n "$clean_value" ]; do
                  # Process one character at a time
                  local char="${clean_value:0:1}"
                  clean_value="${clean_value:1}"

                  local escaped_char="$char"
                  if [ "$char" = "\\" ] || [ "$char" = "\\"" ] || [ "$char" = "\\$" ]; then
                      escaped_char="\\\\$char"
                  fi

                  # Check if adding this character exceeds width
                  # we compute the length of: name + "+=\"" + current_chunk + escaped_char + "\""
                  local test_len=$((${#name} + 4 + ${#current_chunk} + ${#escaped_char} + 1))
                  if [ $test_len -gt 80 ] && [ -n "$current_chunk" ]; then
                      echo "${name}+=\\"${current_chunk}\\""
                      current_chunk="$escaped_char"
                  else
                      current_chunk="${current_chunk}${escaped_char}"
                  fi
              done

              if [ -n "$current_chunk" ]; then
                  echo "${name}+=\\"${current_chunk}\\""
              fi
          }
[[ end ]]
"""

content = content.replace("`", "'")

with open(filepath, 'w') as f:
    f.write(content)
