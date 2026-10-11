#!/bin/sh
scan_memory() {
    MEM_TOTAL=unknown; MEM_AVAILABLE=unknown; MEM_SWAP_TOTAL=unknown; MEM_SWAP_FREE=unknown; MEM_CGROUP_LIMIT=unknown
    _m=$(path /proc/meminfo); if [ -r "$_m" ]; then while read -r _k _v _rest; do case $_k in MemTotal:) MEM_TOTAL=$((_v*1024));; MemAvailable:) MEM_AVAILABLE=$((_v*1024));; SwapTotal:) MEM_SWAP_TOTAL=$((_v*1024));; SwapFree:) MEM_SWAP_FREE=$((_v*1024));; esac; done < "$_m"; fi
    for _f in /sys/fs/cgroup/memory.max /sys/fs/cgroup/memory/memory.limit_in_bytes; do _x=$(read_one "$_f"); case $_x in ''|max|unknown|9223372036854771712) :;; *[!0-9]*) :;; *) MEM_CGROUP_LIMIT=$_x; break;; esac; done
}
memory_json() { printf '{"physical_total_bytes":'; json_quote "$MEM_TOTAL"; printf ',"available_bytes":'; json_quote "$MEM_AVAILABLE"; printf ',"swap_total_bytes":'; json_quote "$MEM_SWAP_TOTAL"; printf ',"swap_available_bytes":'; json_quote "$MEM_SWAP_FREE"; printf ',"cgroup_limit_bytes":'; json_quote "$MEM_CGROUP_LIMIT"; printf ',"evidence":"/proc/meminfo and cgroup files"}'; }
