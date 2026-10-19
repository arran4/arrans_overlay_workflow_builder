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

              if [ -z "$raw_val" ]; then
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

                  if [ $first_word -eq 1 ]; then
                      if [ $((word_len)) -le $available ]; then
                          current_line+="${word}"
                          available=$((available - word_len))
                          first_word=0
                      else
                          # Can't even fit the first word, close the line and start a new one if possible, otherwise just append and break the limit (can't break words)
                          local available_new=$((80 - ${#var_name} - 4 - 1))
                          if [ $prefix_len -gt $((${#var_name} + 3)) ]; then
                              # We are already on a continuation, but it's empty so it shouldn't be long. Just put it.
                              current_line+="${word}"
                              available=$((available - word_len))
                              first_word=0
                          else
                              current_line+="${word}"
                              available=$((available - word_len))
                              first_word=0
                          fi
                      fi
                  else
                      if [ $((word_len + 1)) -le $available ]; then
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

              if [ $first_word -eq 0 ]; then
                  echo "${current_line}\\""
              fi
          }
[[ end ]]
"""

# Let's write the character-based chunking that was given to us originally, which is completely width-safe.
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

              local prefix="${name}+=\\" "
              local suffix=" \\""

              local escaped_value="${value//\\\\/\\\\\\\\}"
              escaped_value="${escaped_value//\\"/\\\\\\"}"
              escaped_value="${escaped_value//\\$/\\\\\\$}"
              escaped_value="${escaped_value//$`\\r`/}"
              escaped_value="${escaped_value//$`\\n`/ }"
              escaped_value="${escaped_value//$`\\t`/ }"

              # Now we do the actual width calculation, making sure we account for expansion width if needed,
              # but just using bash parameter substitution lengths.

              # Oh, wait. The problem in the test is that we expected:
              # 123456789012345678901234567890123456789012345678901234567890123${P}
              # If we chunk it arbitrarily mid-word, we might break ${P} apart!
              # Wait, the user specifically says: "The fixture places a literal ${P} across the 65-column DESCRIPTION chunk boundary. Check semantic reconstruction, not just line widths."

              # "Use proper shell escaping for each emitted fragment. Choose chunk boundaries based on the actual rendered assignment-line width, including: variable name, \+=, shell escaping expansion. Do not use a fixed "raw 80 characters" assumption. Reject or deliberately handle CR/LF/tab input rather than silently embedding it."

              # I should implement chunking, but how do I get the length of the string to split it?
"""
