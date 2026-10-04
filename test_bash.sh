#!/bin/bash
emit_metadata_field() {
    local label=$1
    local value=$2
    local prefix="${label}+=\""
    local width

    echo "${label}=\"\""
    while [ -n "$value" ]; do
        width=$((80 - ${#prefix} - 1)) # -1 for closing quote
        if ((${#value} <= width)); then
            printf '%s%s\"\n' "$prefix" "$value"
            break
        fi

        printf '%s%s\"\n' "$prefix" "${value:0:width}"
        value="${value:width}"
    done
}

emit_metadata_field "DESCRIPTION" "This is a very long description that should be split into multiple assignments using bash += syntax so it remains less than 80 chars per line."
