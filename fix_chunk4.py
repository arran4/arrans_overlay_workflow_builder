import re

filepath = "templates/_partials/ebuild_output_helpers.tmpl"
with open(filepath, 'r') as f:
    content = f.read()

new_chunk_logic = """
          emit_ebuild_assignment() {
              local var_name="$1"
              local raw_val="$2"

              if [[ -z "$raw_val" ]]; then
                  echo "${var_name}=\"\""
                  return 0
              fi

              local -a words
              read -ra words <<< "$raw_val"

              echo "${var_name}=\"\""

              local current_line="${var_name}+=\""
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
                          echo "${current_line}\""
                          current_line="${var_name}+=\" ${word}"
                          prefix_len=${#var_name}
                          available=$((80 - prefix_len - 4 - word_len - 1))
                          first_word=0
                      fi
                  fi
              done

              if [[ $first_word -eq 0 ]]; then
                  echo "${current_line}\""
              fi
          }
"""

content = re.sub(r'emit_ebuild_assignment\(\) \{.*?(?=\n\s*\n|\Z|\n\[\[)', new_chunk_logic.strip('\n'), content, flags=re.DOTALL)

with open(filepath, 'w') as f:
    f.write(content)
