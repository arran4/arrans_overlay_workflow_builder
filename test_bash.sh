#!/bin/bash
name="DESCRIPTION"
current_chunk='foo ${P} bar'
echo "${name}+=\"${current_chunk}\""
