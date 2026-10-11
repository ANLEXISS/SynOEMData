#!/bin/sh
# Synora Runtime Foundation host discovery. Read-only except for --output.
set -u
BOOTSTRAP_VERSION=0.1.0
BOOTSTRAP_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd) || exit 70
BOOTSTRAP_ROOT=${SYNORA_BOOTSTRAP_ROOT:-}
for _lib in json os cpu memory storage network gpu npu peripherals processes environment security dependencies; do
    . "$BOOTSTRAP_DIR/lib/$_lib.sh" || exit 70
done

usage() {
    cat <<'EOF'
Usage: bootstrap.sh [--scan|--json|--summary|--validate] [--output DIRECTORY]
  --scan       Perform a read-only host scan (default)
  --json       Write the normalized profile to stdout
  --output DIR Write synora-host-profile.json in DIR
  --summary    Print a concise human-readable summary
  --validate   Scan and validate mandatory profile fields
  --help       Show this help
EOF
}
mode=scan; output_dir=
while [ "$#" -gt 0 ]; do
    case $1 in
        --scan|--json|--summary|--validate) mode=${1#--};;
        --output) shift; [ "$#" -gt 0 ] || { echo 'missing directory after --output' >&2; exit 64; }; output_dir=$1;;
        --help) usage; exit 0;;
        *) echo "unknown option: $1" >&2; usage >&2; exit 64;;
    esac
    shift
done

scan_all() {
    scan_os; scan_cpu; scan_memory; scan_storage; scan_network; scan_gpu
    scan_npu; scan_peripherals; scan_processes; scan_environment; scan_security; scan_dependencies
}
emit_profile() {
    printf '{"schema_version":"1.0","bootstrap_version":"%s","timestamp_utc":' "$BOOTSTRAP_VERSION"
    json_quote "$(utc_timestamp)"; printf ',"host":'; host_json
    printf ',"os":'; os_json; printf ',"cpu":'; cpu_json; printf ',"memory":'; memory_json
    printf ',"storage":'; storage_json; printf ',"network":'; network_json; printf ',"gpu":'; gpu_json
    printf ',"npu":'; npu_json; printf ',"peripherals":'; peripherals_json
    printf ',"process_management":'; processes_json; printf ',"execution_environment":'; environment_json
    printf ',"security":'; security_json; printf ',"dependencies":'; dependencies_json
    printf ',"detection_errors":[]}'
}
host_json() {
    printf '{"hostname":'; json_quote "$(read_one /etc/hostname)"; printf ',"status":"%s","value":"%s","evidence":"uname"}' detected "$(uname -n 2>/dev/null || printf unknown)"
}
validate_profile() {
    case ${OS_KERNEL:-} in ''|unknown) echo 'missing mandatory field: os.kernel' >&2; return 65;; esac
    case ${CPU_ARCH:-} in ''|unknown) echo 'missing mandatory field: cpu.architecture' >&2; return 65;; esac
    return 0
}
summary() {
    printf 'Synora host profile %s\nOS: %s (%s)\nArchitecture: %s\nInit/service manager: %s / %s\nExecution: %s (%s)\nMemory: %s bytes physical, %s bytes available\nGPU: %s\nNPU: %s\n' \
        "$BOOTSTRAP_VERSION" "${OS_ID:-unknown}" "${OS_VERSION:-unknown}" "${CPU_ARCH:-unknown}" \
        "${PID1_KIND:-unknown}" "${SERVICE_MANAGER:-unknown}" "${ENV_KIND:-unknown}" "${ENV_RUNTIME:-unknown}" \
        "${MEM_TOTAL:-unknown}" "${MEM_AVAILABLE:-unknown}" "${GPU_FAMILY:-unknown}" "${NPU_ECOSYSTEM:-unknown}"
}
scan_all
if [ -n "$output_dir" ]; then
    [ -d "$output_dir" ] && [ -w "$output_dir" ] || { echo "invalid or unwritable output directory: $output_dir" >&2; exit 73; }
    emit_profile > "$output_dir/synora-host-profile.json" || { echo 'failed to write profile' >&2; exit 70; }
fi
case $mode in
    json) emit_profile; printf '\n'; [ "$OS_FAMILY" != unsupported ] || exit 2;;
    summary) summary;;
    validate) validate_profile || exit $?; emit_profile; printf '\n'; [ "$OS_FAMILY" != unsupported ] || exit 2;;
    scan) [ -n "$output_dir" ] || summary; [ "$OS_FAMILY" != unsupported ] || exit 2;;
esac
