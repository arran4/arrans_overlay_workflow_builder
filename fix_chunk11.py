filepath = "templates/_partials/ebuild_output_helpers.tmpl"

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

              local prefix="${name}+=\\""
              local suffix="\\""

              # First clean up whitespace
              local escaped_value="${value//$`\\r`/}"
              escaped_value="${escaped_value//$`\\n`/ }"
              escaped_value="${escaped_value//$`\\t`/ }"

              local current_value="$escaped_value"
              local output=""
              local current_chunk=""

              while [ -n "$current_value" ]; do
                  # Process one character at a time
                  local char="${current_value:0:1}"
                  current_value="${current_value:1}"

                  local escaped_char="$char"
                  if [ "$char" = "\\" ] || [ "$char" = "\\"" ] || [ "$char" = "\\$" ]; then
                      escaped_char="\\\\$char"
                  fi

                  # Check if adding this character exceeds width
                  # We use a dummy evaluation here to check length
                  local test_str="${name}+=\\"${current_chunk}${escaped_char}\\""
                  if [ ${#test_str} -gt 80 ] && [ -n "$current_chunk" ]; then
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
