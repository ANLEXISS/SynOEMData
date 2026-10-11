# Source-to-OEM extraction map

## Mapping rule

The copy preserves each included source-relative path exactly beneath the OEM
root. Thus every included source path `P` maps to
`/home/rock/SynoraOEM/P`. This preserves `module synora`, Go imports,
Python package-relative assumptions, CMake paths, fixture references and
service/config paths. Domain ownership is the one in
[`02-oem-module-map.md`](02-oem-module-map.md); a file is not duplicated when a
responsibility crosses domains.

| Original path prefix | OEM path prefix | Domain / status |
| --- | --- | --- |
| `/home/rock/Synora-v1/go.mod`, `go.sum` | `/home/rock/SynoraOEM/go.mod`, `go.sum` | Single Go module, preserved verbatim. |
| `cmd/**` | `cmd/**` | Service, CLI, benchmark and test entrypoints; preserved. |
| `internal/**` | `internal/**` | Private Go implementation packages. M1.2 moved Vision from `internal/discovery/vision` to `internal/vision`; imports were updated in this OEM copy only. |
| `pkg/contract/**`, `pkg/contracts/**` | same relative paths | Shared wire/domain contracts; preserved. |
| `services/vision-worker/**` | same relative paths | Python perception worker/tests; preserved; Python bytecode excluded. |
| `synora-web/**` | same relative paths | UI source and npm lockfiles; preserved; `node_modules` and generated `dist` excluded. |
| `camera/v0/**` | same relative paths | Pre-existing untracked C++ simulation/mock source; preserved; nested build artifacts excluded. |
| `bootstrap/**` | same relative paths | Pre-existing untracked POSIX host discovery source/tests; preserved. |
| `configs/**`, `deployments/**` | same relative paths | Current config examples/templates and Linux deployment descriptors; preserved. No secret material was found or copied. |
| `docs/**`, `testdata/**`, `tools/**` | same relative paths | Product documentation, fixtures, qualification and audit tools; preserved. Large recordings/models are excluded by type and were not tracked. |
| `Makefile`, `.gitignore`, `CHANGELOG_V1.md`, `SYNORA_V1_PROGRESS_AUDIT.md` | same relative paths | Build/config/history/audit; copied as source context. OEM root `README.md` was replaced with a workspace-specific README. |

At extraction time 537 tracked plus 52 ordinary untracked source files were
present. All 589 were copied before adding OEM documentation; generated web
`dist` was removed from the initial archive. Sixteen additional small JSON
files were copied from `build/cognitive-mlp-v1/` and
`build/cognitive-mlp-v3-candidate/` (496 KB combined). Go tests and
`cmd/synora-central-test` load these model bundles from fixed relative paths.
They declare dry-run model execution and contain JSON weights/reports, not
compiled executables. They are the only retained items from the otherwise
excluded `build/` tree. The current target contains those source/resource
files plus the OEM README and architecture documents.

## Excluded from the copy

| Original content | Reason / disposition |
| --- | --- |
| `.git/` | No history/remotes/commit copied; workspace is independent. Source branch/HEAD recorded in audit. |
| `build/**` except `build/cognitive-mlp-v1/**` and `build/cognitive-mlp-v3-candidate/**`, `bin/`, `camera/v0/build/` | Compiled outputs, generated datasets/model artifacts and local build cache. Rebuild from source. The two JSON bundles are retained because existing tests hard-code those resource paths. |
| `synora-web/node_modules/`, `synora-web/dist/` | Installed platform-specific dependencies and generated UI output. Reinstall from lockfile and rebuild. |
| `__pycache__/`, `*.pyc`, virtual environments | Runtime/cache output. Dependencies remain documented in requirements. |
| Model/recording/credential extensions and `secrets/`, `.env*` | Explicitly excluded; none were in tracked/untracked source inputs. External models/secrets must be supplied in a controlled deployment. |

The original source remains at `/home/rock/Synora-v1` and must remain read-only
for OEM development. Re-run the file mapping inventory whenever either tree
changes; this report records the initial extraction, not future sync behavior.

The M1.2 source move and module ownership are tracked in
[`modules/package-ownership.json`](../../modules/package-ownership.json), with
validation in [`16-modularization-validation.md`](16-modularization-validation.md).
