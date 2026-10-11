#!/bin/sh
scan_os() {
    OS_KERNEL=$(uname -s 2>/dev/null || printf unknown); OS_RELEASE=$(uname -r 2>/dev/null || printf unknown)
    OS_KARCH=$(uname -m 2>/dev/null || printf unknown); OS_ID=unknown; OS_VERSION=unknown; OS_FAMILY=other_linux; OS_LIBC=unknown
    _osfile=$(path /etc/os-release); [ -r "$_osfile" ] || _osfile=$(path /usr/lib/os-release)
    if [ -r "$_osfile" ]; then
        while IFS= read -r _line || [ -n "$_line" ]; do
            case $_line in ID=*) OS_ID=$(parse_os_value "${_line#ID=}");;
                VERSION_ID=*) OS_VERSION=$(parse_os_value "${_line#VERSION_ID=}");; esac
        done < "$_osfile"
    fi
    case $OS_ID in *[!A-Za-z0-9._-]*|'') OS_ID=unknown;; esac
    case $OS_VERSION in *[!A-Za-z0-9._+-]*) OS_VERSION=unknown;; esac
    case $OS_ID in debian|ubuntu|raspbian|linuxmint) OS_FAMILY=debian;; rhel|fedora|centos|rocky|almalinux) OS_FAMILY=rhel;; arch|manjaro) OS_FAMILY=arch;; alpine) OS_FAMILY=alpine;; openwrt) OS_FAMILY=openwrt;; buildroot) OS_FAMILY=buildroot;; poky|yocto) OS_FAMILY=yocto;; android) OS_FAMILY=android;; ''|unknown) OS_ID=unknown; OS_FAMILY=other_linux;; esac
    case $OS_KERNEL in Linux) :;; *) OS_FAMILY=unsupported;; esac
    if has_cmd getconf; then _clib=$(getconf GNU_LIBC_VERSION 2>/dev/null || :); case $_clib in glibc*) OS_LIBC=glibc;; esac; fi
    [ "$OS_LIBC" != unknown ] || { [ -e "$(path /lib/ld-musl-aarch64.so.1)" ] || [ -e "$(path /lib/ld-musl-x86_64.so.1)" ] || [ -e "$(path /lib/ld-musl-armhf.so.1)" ] && OS_LIBC=musl; }
}
parse_os_value() {
    _value=$1
    case $_value in
        \"*) case $_value in *\") _value=${_value#\"}; _value=${_value%\"};; *) _value=unknown;; esac;;
        \'*) case $_value in *\') _value=${_value#\'}; _value=${_value%\'};; *) _value=unknown;; esac;;
    esac
    printf '%s' "$_value"
}
os_json() {
    printf '{"kernel":'; status_value_json detected "$OS_KERNEL" uname
    printf ',"kernel_release":'; json_quote "$OS_RELEASE"; printf ',"kernel_architecture":'; json_quote "$OS_KARCH"
    printf ',"distribution_id":'; status_value_json "$( [ "$OS_ID" = unknown ] && echo unknown || echo detected )" "$OS_ID" "${_osfile:-os-release unavailable}"
    printf ',"distribution_version":'; json_quote "$OS_VERSION"; printf ',"distribution_family":'; json_quote "$OS_FAMILY"
    printf ',"operating_system_id":'; json_quote "$OS_KERNEL:$OS_ID"; printf ',"linux_abi_hints":[] ,"libc_family":'; json_quote "$OS_LIBC"; printf '}'
}
