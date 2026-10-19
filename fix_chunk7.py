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
              local var_name="$1"
              local raw_val="$2"

              if [[ -z "$raw_val" ]]; then
                  echo "${var_name}=\\"\\""
                  return 0
              fi

              local -a words
              read -ra words <<< "$raw_val"

              echo "${var_name}=\\"\\""

              local current_line="${var_name}+=\\""
              local prefix_len=${#current_line}
              local available=$((80 - prefix_len - 1))

              local first_word=1

              for word in "${words[@]}"; do
                  local word_len=${#word}

                  if [[ $first_word -eq 1 ]]; then
                      current_line+="${word}"
                      available=$((available - word_len))
                      first_word=0
                  else
                      if [[ $((word_len + 1)) -le $available ]]; then
                          current_line+=" ${word}"
                          available=$((available - word_len - 1))
                      else
                          echo "${current_line}\\""
                          current_line="${var_name}+=\\" ${word}"
                          prefix_len=${#var_name}
                          available=$((80 - prefix_len - 4 - word_len - 1))
                          first_word=0
                      fi
                  fi
              done

              if [[ $first_word -eq 0 ]]; then
                  echo "${current_line}\\""
              fi
          }
[[ end ]]
"""

with open(filepath, 'w') as f:
    f.write(content)
