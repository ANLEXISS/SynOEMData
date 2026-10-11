#!/bin/sh
set -eu
HERE=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
BOOT=$HERE/bootstrap.sh
TMP=${TMPDIR:-/tmp}/synora-bootstrap-test.$$
trap 'rm -rf "$TMP"' EXIT HUP INT TERM
mkdir -p "$TMP"
fail() { echo "FAIL: $*" >&2; exit 1; }
pass() { echo "PASS: $*"; }

for shell in dash "busybox ash" "bash --posix"; do
    if command -v "${shell%% *}" >/dev/null 2>&1; then
        # Parsing by each available POSIX-compatible shell is a static check.
        set -- $shell
        "$@" -n "$BOOT" || fail "$shell syntax"
        pass "$shell syntax"
    fi
done
if command -v bash >/dev/null 2>&1; then
    ln -s "$(command -v bash)" "$TMP/sh"
    "$TMP/sh" -n "$BOOT" || fail 'bash invoked as sh syntax'
    pass 'bash invoked as sh syntax'
fi

check_fixture() {
    label=$1; expected=$2; manager=$3; fixture=$HERE/tests/fixtures/$label
    profile=$(SYNORA_BOOTSTRAP_ROOT="$fixture" "$BOOT" --json) || fail "$label scan"
    printf '%s' "$profile" | grep -q '"schema_version":"1.0"' || fail "$label schema marker"
    if command -v python3 >/dev/null 2>&1; then printf '%s' "$profile" | python3 -c 'import json,sys; d=json.load(sys.stdin); assert d["schema_version"] == "1.0"; assert d["npu"]["inference_verified"] is False; assert d["os"]["distribution_id"]["value"] == sys.argv[1]; assert d["process_management"]["service_manager"] == sys.argv[2]' "$expected" "$manager" || fail "$label profile assertions"; else
        printf '%s' "$profile" | grep -q "\"distribution_id\":{\"status\":\"detected\",\"value\":\"$expected\"" || fail "$label distribution"
        printf '%s' "$profile" | grep -q "\"service_manager\":\"$manager\"" || fail "$label service manager"
    fi
    pass "$label fixture"
}
check_fixture debian-systemd debian systemd
check_fixture openwrt-procd openwrt procd
check_fixture alpine-openrc alpine openrc
check_fixture buildroot-busybox buildroot busybox_init
check_fixture container-external unknown external
check_fixture unknown-linux vendor_custom_linux unknown
check_fixture malformed-os unknown unknown
check_fixture unknown-manager custom unknown
check_fixture missing-proc debian unknown
check_fixture restricted-sys alpine unknown

profile=$(SYNORA_BOOTSTRAP_ROOT="$HERE/tests/fixtures/rockchip-rk3588" "$BOOT" --json)
printf '%s' "$profile" | grep -q '"soc":"rk3588"' || fail 'RK3588 evidence'
printf '%s' "$profile" | grep -q '"inference_verified":false' || fail 'inference must remain unverified'
pass 'Rockchip RK3588 conservative detection'

profile=$(SYNORA_BOOTSTRAP_ROOT="$HERE/tests/fixtures/restricted-sys" "$BOOT" --json)
printf '%s' "$profile" | grep -q '"device_nodes":""' || fail 'no GPU false positive'
printf '%s' "$profile" | grep -q '"hardware_detected":false' || fail 'no NPU false positive'
pass 'no GPU and no NPU fixture'

# Minimal PATH deliberately omits ip, lscpu, nproc, df, getconf, ldconfig,
# lspci, systemd utilities, and all optional runtime dependencies.
MINPATH=$TMP/minimal-path
mkdir "$MINPATH"
for util in awk sed grep tr date uname id cat dirname readlink; do
    if command -v "$util" >/dev/null 2>&1; then ln -s "$(command -v "$util")" "$MINPATH/$util"; fi
done
profile=$(PATH="$MINPATH" "$BOOT" --json) || fail 'scan with optional commands absent'
if command -v python3 >/dev/null 2>&1; then printf '%s' "$profile" | python3 -c 'import json,sys; d=json.load(sys.stdin); assert d["network"]["ip_command_available"] is False' || fail 'missing-utilities JSON parse'; fi
pass 'missing optional utilities'

profile=$(SYNORA_BOOTSTRAP_ROOT="$HERE/tests/fixtures/container-external" "$BOOT" --json)
printf '%s' "$profile" | grep -q '"distribution_family":"other_linux"' || fail 'unknown Linux fallback'
pass 'container and generic Linux scan'

"$BOOT" --validate >/dev/null || fail 'real-host validation'
"$BOOT" --summary >/dev/null || fail 'real-host smoke scan'
out=$TMP/report
mkdir "$out"
"$BOOT" --output "$out" --scan || fail 'output write'
[ -s "$out/synora-host-profile.json" ] || fail 'output report missing'
pass 'real-host smoke and output write'
echo 'All bootstrap tests passed.'
