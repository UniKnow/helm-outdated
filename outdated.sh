#! /usr/bin/env bash

set -euo pipefail

while IFS= read -r dependency; do
    subchart="${dependency%-*}"
    current="${dependency##*-}"
    latest=$(helm inspect chart "$subchart" | yq -r '.version')

    if [ "$current" == "$latest" ]; then
        printf "%s is up to date.\\n" "$subchart"
    else
        printf "Consider upgrading %s: %s -> %s.\\n" \
            "$subchart" "$current" "$latest"
    fi
done < <(helm dep list \
    | grep -v WARNING \
    | tail -n+2 | head -n-1 | sort -u \
    | awk '{ sub("@","",$3); printf "%s/%s-%s\n", $3, $1, $2; }')
