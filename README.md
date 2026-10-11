# Synora OEM workspace

This is an isolated working copy of the Synora V1 source tree for OEM
architecture work. It keeps one Go module and existing process entrypoints.
M1.2 moved Vision from `internal/discovery/vision` to `internal/vision` and
topology from `internal/topology` to `internal/intelligence/topology`. It added
injected lifecycle and network station-source interfaces. Worker protocol,
process entrypoints, and runtime behavior are preserved.

The architecture inventory and M1.2 audit are in
[`docs/architecture/`](docs/architecture/). Start with
[`11-module-boundaries.md`](docs/architecture/11-module-boundaries.md),
[`16-modularization-validation.md`](docs/architecture/16-modularization-validation.md),
and the machine-readable [`modules/manifest.json`](modules/manifest.json).

The copy includes the source repository's pre-existing working-tree changes in
`Makefile`, `bootstrap/`, and `camera/`. Git metadata, build outputs, binaries,
Python bytecode, node modules, generated web output, and large model/video
artifacts are excluded. Two small JSON-only V1/V3 dry-run bundles (496 KB
total) are retained at their original paths because Go tests and the central
harness directly require them. No Git remote or commit was created.

The module registry describes existing code and labels absent Voice and
Automation implementations as not implemented. `make architecture-check`
checks descriptors and Go package boundaries; `make test` includes that check.
No OEM adapters, new algorithms, dynamic plugin loading, or service split were
implemented. The deployment remains Linux/systemd-specific; other host
integrations are future work documented in the roadmap.

The original source repository is `/home/rock/Synora-v1`. Treat it as read-only
while working in this OEM copy.
