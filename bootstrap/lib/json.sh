#!/bin/sh
# POSIX awk JSON string quoting. Input is kept in a single argument; controls
# are escaped before emission, so observed system strings cannot break JSON.
json_quote() {
    printf '"'; printf '%s' "${1-}" | awk 'BEGIN { ORS="" } { if (NR>1) printf "\\n"; gsub(/\\/,"\\\\"); gsub(/"/,"\\\""); gsub(sprintf("%c",8),"\\b"); gsub(sprintf("%c",9),"\\t"); gsub(sprintf("%c",12),"\\f"); gsub(sprintf("%c",13),"\\r"); printf "%s", $0 }'; printf '"'
}
utc_timestamp() { date -u '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || printf unknown; }
path() { printf '%s%s' "$BOOTSTRAP_ROOT" "$1"; }
read_one() { [ -r "$(path "$1")" ] && { IFS= read -r _v < "$(path "$1")" || :; printf '%s' "${_v-}"; } || :; }
has_cmd() { command -v "$1" >/dev/null 2>&1; }
status_value_json() { printf '{"status":'; json_quote "$1"; printf ',"value":'; json_quote "$2"; printf ',"evidence":'; json_quote "$3"; [ -z "${4-}" ] || { printf ',"error":'; json_quote "$4"; }; printf '}'; }
