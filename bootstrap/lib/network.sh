#!/bin/sh
scan_network() { :; }
network_json() {
    printf '{"interfaces":['; _comma=
    _net=$(path /sys/class/net)
    if [ -d "$_net" ]; then
        for _i in "$_net"/*; do
            [ -e "$_i" ] || continue
            _n=${_i##*/}; _t=unknown; [ "$_n" = lo ] && _t=loopback
            [ -d "$_i/wireless" ] && _t=wireless
            if [ -e "$_i/type" ] && [ "$(cat "$_i/type" 2>/dev/null)" = 1 ] && [ "$_n" != lo ]; then _t=ethernet; fi
            _state=$(cat "$_i/operstate" 2>/dev/null || printf unknown)
            _mac=$(cat "$_i/address" 2>/dev/null || printf unknown)
            _v4=unknown; _v6=unknown
            if has_cmd ip; then
                _v4=$(ip -o -4 address show dev "$_n" 2>/dev/null | awk '{for(i=1;i<=NF;i++) if($i=="inet") print $(i+1)}' | tr '\n' ' ')
                _v6=$(ip -o -6 address show dev "$_n" 2>/dev/null | awk '{for(i=1;i<=NF;i++) if($i=="inet6") print $(i+1)}' | tr '\n' ' ')
                [ -n "$_v4" ] || _v4=unavailable; [ -n "$_v6" ] || _v6=unavailable
            fi
            printf '%s{"name":' "$_comma"; _comma=,
            json_quote "$_n"; printf ',"type":'; json_quote "$_t"; printf ',"operational_state":'; json_quote "$_state"
            printf ',"mac_address":'; json_quote "$_mac"; printf ',"ipv4_addresses":'; json_quote "$_v4"; printf ',"ipv6_addresses":'; json_quote "$_v6"; printf '}'
        done
    fi
    printf '],"ip_command_available":%s,"active_scanning":false}' "$(has_cmd ip && echo true || echo false)"
}
