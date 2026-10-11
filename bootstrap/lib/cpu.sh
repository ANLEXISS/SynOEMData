#!/bin/sh
scan_cpu() {
    _a=$(uname -m 2>/dev/null || printf unknown); case $_a in aarch64|arm64) CPU_ARCH=arm64;; arm*|armv*) CPU_ARCH=arm32;; x86_64|amd64) CPU_ARCH=x86_64;; i?86) CPU_ARCH=x86;; riscv64) CPU_ARCH=riscv64;; mips*) CPU_ARCH=mips;; unknown|'') CPU_ARCH=unknown;; *) CPU_ARCH=other;; esac
    CPU_MODEL=unknown; CPU_VENDOR=unknown; CPU_COUNT=unknown; CPU_CORES=unknown; CPU_FREQ=unknown; CPU_FLAGS=; CPU_PLATFORM=unknown
    _cpu=$(path /proc/cpuinfo)
    if [ -r "$_cpu" ]; then
        while IFS= read -r _l; do case $_l in *:*) _k=${_l%%:*}; _v=${_l#*:}; _v=$(printf '%s' "$_v" | sed 's/^[[:space:]]*//'); case $_k in model\ name|Processor|Hardware|cpu\ model) [ "$CPU_MODEL" != unknown ] || CPU_MODEL=$_v;; vendor_id|CPU\ implementer) [ "$CPU_VENDOR" != unknown ] || CPU_VENDOR=$_v;; flags|Features) [ -n "$CPU_FLAGS" ] || CPU_FLAGS=$_v;; esac;; esac; done < "$_cpu"
    fi
    if [ -d "$(path /sys/devices/system/cpu)" ]; then
        CPU_COUNT=0
        for _c in "$(path /sys/devices/system/cpu)"/cpu[0-9]*; do [ -d "$_c" ] && CPU_COUNT=$((CPU_COUNT+1)); done
        [ "$CPU_COUNT" -gt 0 ] 2>/dev/null || CPU_COUNT=unknown
    elif has_cmd nproc; then CPU_COUNT=$(nproc 2>/dev/null || printf unknown); fi
    CPU_PLATFORM=$(read_one /sys/firmware/devicetree/base/model); [ -n "$CPU_PLATFORM" ] || CPU_PLATFORM=unknown
    if [ -r "$(path /sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq)" ]; then CPU_FREQ=$(($(read_one /sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq) * 1000)) 2>/dev/null || CPU_FREQ=unknown; fi
}
cpu_json() { printf '{"architecture":'; status_value_json "$( [ "$CPU_ARCH" = unknown ] && echo unknown || echo detected )" "$CPU_ARCH" uname; printf ',"model":'; json_quote "$CPU_MODEL"; printf ',"vendor":'; json_quote "$CPU_VENDOR"; printf ',"logical_cpu_count":'; json_quote "$CPU_COUNT"; printf ',"core_count":'; json_quote "$CPU_CORES"; printf ',"frequency_hz":'; json_quote "$CPU_FREQ"; printf ',"instruction_set_hints":'; json_quote "$CPU_FLAGS"; printf ',"hardware_platform":'; json_quote "$CPU_PLATFORM"; printf '}' ; }
