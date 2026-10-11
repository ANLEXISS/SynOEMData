#!/bin/sh
scan_storage() {
    STORAGE_ROOT_FS=unknown; STORAGE_ROOT_MODE=unknown; STORAGE_CAPACITY=unknown; STORAGE_AVAILABLE=unknown; STORAGE_OVERLAY=false; STORAGE_DEVICES=
    _mount=$(path /proc/mounts); if [ -r "$_mount" ]; then while read -r _src _dst _fs _opts _rest; do if [ "$_dst" = / ]; then STORAGE_ROOT_FS=$_fs; case ,$_opts, in *,ro,*) STORAGE_ROOT_MODE=read_only;; *,rw,*) STORAGE_ROOT_MODE=read_write;; esac; case $_fs in overlay) STORAGE_OVERLAY=true;; esac; break; fi; done < "$_mount"; fi
    if has_cmd df; then _df=$(df -Pk / 2>/dev/null | awk 'NR==2 {print $2, $4}'); set -- $_df; [ $# -ge 2 ] && { STORAGE_CAPACITY=$(($1*1024)); STORAGE_AVAILABLE=$(($2*1024)); }; fi
    if [ -d "$(path /sys/block)" ]; then for _b in "$(path /sys/block)"/*; do [ -e "$_b" ] || continue; STORAGE_DEVICES="$STORAGE_DEVICES${_b##*/} "; done; fi
}
storage_json() { printf '{"root_filesystem_type":'; json_quote "$STORAGE_ROOT_FS"; printf ',"root_access_mode":'; json_quote "$STORAGE_ROOT_MODE"; printf ',"root_capacity_bytes":'; json_quote "$STORAGE_CAPACITY"; printf ',"root_available_bytes":'; json_quote "$STORAGE_AVAILABLE"; printf ',"block_devices":'; json_quote "$STORAGE_DEVICES"; printf ',"overlay_filesystem":%s,"persistent_storage_hint":"unknown","container_visibility_limitations":"may be namespace-scoped"}' "$STORAGE_OVERLAY"; }
