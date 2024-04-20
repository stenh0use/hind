#!/usr/bin/env bash

set -o pipefail

images=("$@")
missing=()

for image in "${images[@]}"; do
    count=$(docker image inspect "${image}:${HIND_VERSION}" | jq ".|length")
    if [ "${count:-0}" -eq 0 ]; then
        missing+=("${image/\:[0-9]*.[0-9]*.[0-9]*/}")
    fi
done

printf "%s\n" "${missing[@]}"
