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

RDEPEND="sys-libs/glibc sys-libs/zlib $(echo -n 'some? ( other ) ')"
emit_metadata_field "RDEPEND" "$RDEPEND"
