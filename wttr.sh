#! /bin/bash
select_random() {
    printf "%s\0" "$@" | shuf -z -n1 | tr -d '\0'
}

expressions=("Montreal" "Victoria" "Calgary" "Toronto" "Quebec")
selectedexpression=$(select_random "${expressions[@]}")
curl -s "wttr.in/$selectedexpression?0m"

