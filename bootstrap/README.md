# Synora Bootstrap 0.1.0

`bootstrap.sh` performs a read-only Linux host survey for the future Synora
Runtime Foundation. It does not install software, contact the network, start
services, run inference, or change host configuration. The only write is the
profile explicitly requested with `--output`, and it is written only inside
the supplied existing directory.

## Run it

```sh
./bootstrap/bootstrap.sh --scan
./bootstrap/bootstrap.sh --summary
./bootstrap/bootstrap.sh --json
./bootstrap/bootstrap.sh --output /var/tmp/synora-report
./bootstrap/bootstrap.sh --validate
```

`--json` writes only the profile to standard output; diagnostics use standard
error. `--scan` is the default and prints a compact summary unless `--output`
is supplied. `--validate` checks the mandatory kernel and normalized CPU
architecture fields, then emits the profile.

Exit codes: `0` success, `64` command line error, `65` missing mandatory scan
data, `70` internal failure, and `73` invalid/unwritable output directory.

## Runtime requirements

The script and libraries use POSIX shell and standard utilities commonly
present in a minimal Linux userland: `awk`, `sed`, `grep`, `tr`, `date`,
`uname`, `id`, `cat`, and `dirname`. Missing optional tools do not stop scanning.
Optional evidence sources include `nproc`, `ip`, `df`, `getconf`, `ldconfig`,
`lscpu`, and `lspci`; the first release uses `nproc`, `ip`, `df`, `getconf`,
and `ldconfig` only when present. Python, Bash, jq, curl, systemd, and network
connectivity are not runtime requirements.

## Profile and normalization

The output follows [`schema/host-profile.schema.json`](schema/host-profile.schema.json),
with `schema_version` `1.0`. Unknown scalar values are represented by the
string `"unknown"`; unavailable optional capabilities are represented as
`unavailable` or `false` where the field is explicitly boolean. Each top-level
section is stable and carries its evidence field(s). Architecture names
normalize to `arm64`, `arm32`, `x86_64`, `x86`, `riscv64`, `mips`, `other`, or
`unknown`. Distribution families include Debian, RHEL, Arch, Alpine, OpenWrt,
Buildroot, Yocto, Android, generic Linux, and unsupported non-Linux kernels.
Memory quantities are bytes. RKNN inference is always reported unverified.

The implementation uses conservative filesystem evidence. It does not source
`os-release`; it parses the `ID` and `VERSION_ID` assignments as plain data.
`/etc/os-release` takes precedence over `/usr/lib/os-release`. The platform
classification remains generic Linux if the distribution is unfamiliar.

## Detection coverage and limits

- OS/kernel and libc hints; CPU architecture, model hints, online logical CPU
  count, max frequency, flags, and device-tree platform string.
- Physical and available memory from `/proc/meminfo`, cgroup memory limit,
  root mount type/mode/capacity, visible block devices, and overlay hint.
- Network interface names, link type/state, MAC addresses, and IPv4/IPv6
  addresses when the optional `ip` utility is present.
- DRM nodes and conservative GPU family hints; RK3588 device-tree, RKNPU
  module, RKNN shared library, and relevant device-node hints. A present node
  is only a visibility hint; no compute or inference claim is made.
- PID 1 plus init-file/runtime evidence for common service managers, container
  and VM evidence, a limited set of security signals, and selected dependency
  presence/version hints.
- Peripheral controller hints are based on sysfs class/bus presence; attached
  peripheral usability and access permissions are not probed.

Containers can hide host devices, block devices, cgroup limits, and namespaces.
Unavailable data is not inferred from the absence of one optional command.
The release does not claim complete detection of every supervisor, GPU/NPU
vendor, security policy, ABI, or embedded distribution. `detection_errors` is
reserved as a stable array; optional probe failures currently remain unknown
in their fields instead of adding per-probe entries.

## Tests

Run `./bootstrap/tests/run-tests.sh`. It checks POSIX syntax with available
shells, parses generated JSON with Python 3 when present, uses filesystem
fixtures for common/unknown/malformed distributions and RK3588 evidence, then
runs validation, summary, and output smoke checks on the current host. It does
not require target operating systems or install packages.

Fixture scans can also be run directly with
`SYNORA_BOOTSTRAP_ROOT=bootstrap/tests/fixtures/debian-systemd` to redirect
filesystem evidence into a fixture tree. This is a test hook; kernel
identification and optional host commands still describe the running kernel.
