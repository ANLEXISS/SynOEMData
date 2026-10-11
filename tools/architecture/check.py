#!/usr/bin/env python3
"""Validate the Synora logical-module registry and Go package boundaries."""

from __future__ import annotations

import json
import ast
import os
import re
import subprocess
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
MODULES = ROOT / "modules"
VISION_WORKER = ROOT / "services" / "vision-worker"


def fail(message: str) -> None:
    raise SystemExit(f"architecture check: FAIL: {message}")


def load_json(path: Path) -> dict:
    try:
        return json.loads(path.read_text())
    except (OSError, json.JSONDecodeError) as exc:
        fail(f"cannot read {path.relative_to(ROOT)}: {exc}")


def validate_descriptors() -> tuple[dict[str, dict], dict[str, str]]:
    registry = load_json(MODULES / "manifest.json")
    if registry.get("schema_version") != "synora.module-registry/v1":
        fail("unsupported registry schema version")
    descriptors: dict[str, dict] = {}
    for name in registry.get("modules", []):
        if Path(name).name != name or not name.endswith(".json"):
            fail(f"invalid descriptor filename {name!r}")
        descriptor = load_json(MODULES / name)
        required = {
            "id", "version", "description", "domain", "required_capabilities",
            "optional_capabilities", "required_dependencies", "optional_dependencies",
            "provided_interfaces", "consumed_interfaces", "configuration_schema",
            "state_ownership", "resource_requirements", "maturity", "source_packages",
        }
        missing = required - descriptor.keys()
        if missing:
            fail(f"{name} misses fields: {', '.join(sorted(missing))}")
        module_id = descriptor["id"]
        if module_id in descriptors:
            fail(f"duplicate module id {module_id}")
        if not re.fullmatch(r"synora\.[a-z][a-z0-9.-]*", module_id):
            fail(f"invalid module id {module_id!r}")
        if not re.fullmatch(r"[0-9]+\.[0-9]+\.[0-9]+", descriptor["version"]):
            fail(f"invalid version for {module_id}")
        if descriptor["maturity"] not in {"implemented", "partial", "not_implemented"}:
            fail(f"invalid maturity for {module_id}")
        if set(descriptor["required_capabilities"]) & set(descriptor["optional_capabilities"]):
            fail(f"capability declared required and optional in {module_id}")
        if descriptor["configuration_schema"]:
            config_path = (MODULES / descriptor["configuration_schema"]).resolve()
            if ROOT not in config_path.parents or not config_path.exists():
                fail(f"configuration schema path is missing or escapes workspace: {module_id}")
        for source in descriptor["source_packages"]:
            source_path = ROOT / source
            if not source_path.exists():
                fail(f"declared source path does not exist: {module_id}: {source}")
        descriptors[module_id] = descriptor

    if len(descriptors) != 7:
        fail(f"expected seven logical module descriptors, found {len(descriptors)}")
    graph = {
        module_id: set(desc["required_dependencies"] + desc["optional_dependencies"])
        for module_id, desc in descriptors.items()
    }
    for module_id, dependencies in graph.items():
        missing = dependencies - descriptors.keys()
        if missing:
            fail(f"{module_id} depends on undefined module(s): {', '.join(sorted(missing))}")
    visit: set[str] = set()
    active: list[str] = []

    def walk(node: str) -> None:
        if node in active:
            fail("module dependency cycle: " + " -> ".join(active[active.index(node):] + [node]))
        if node in visit:
            return
        active.append(node)
        for dep in sorted(graph[node]):
            walk(dep)
        active.pop()
        visit.add(node)

    for node in sorted(graph):
        walk(node)
    return descriptors, graph


def validate_composition_profiles(descriptors: dict[str, dict]) -> None:
    profiles_dir = MODULES / "profiles"
    profile_files = sorted(profiles_dir.glob("*.json")) if profiles_dir.exists() else []
    if not profile_files:
        fail("no OEM composition profiles are registered")
    seen: set[str] = set()
    for path in profile_files:
        profile = load_json(path)
        required = {
            "schema_version", "id", "required_modules", "optional_modules",
            "excluded_modules", "runtime_entrypoints", "limitations",
        }
        missing = required - profile.keys()
        if missing:
            fail(f"{path.relative_to(ROOT)} misses fields: {', '.join(sorted(missing))}")
        if profile["schema_version"] != "synora.oem-composition/v1":
            fail(f"unsupported composition profile schema: {path.relative_to(ROOT)}")
        profile_id = profile["id"]
        if not re.fullmatch(r"synora\.oem\.[a-z][a-z0-9.-]*", profile_id) or profile_id in seen:
            fail(f"invalid or duplicate composition profile id: {profile_id!r}")
        seen.add(profile_id)
        selected = set(profile["required_modules"]) | set(profile["optional_modules"])
        excluded = set(profile["excluded_modules"])
        if selected & excluded:
            fail(f"composition profile {profile_id} both selects and excludes a module")
        unknown = (selected | excluded) - descriptors.keys()
        if unknown:
            fail(f"composition profile {profile_id} names unknown modules: {', '.join(sorted(unknown))}")
        for module_id in selected:
            missing_deps = set(descriptors[module_id]["required_dependencies"]) - selected
            if missing_deps:
                fail(f"composition profile {profile_id} omits required dependencies of {module_id}: {', '.join(sorted(missing_deps))}")
        if not profile["runtime_entrypoints"]:
            fail(f"composition profile {profile_id} has no runtime entrypoints")
        for entrypoint in profile["runtime_entrypoints"]:
            if not (ROOT / entrypoint).is_dir():
                fail(f"composition profile {profile_id} entrypoint does not exist: {entrypoint}")
        if not profile["limitations"]:
            fail(f"composition profile {profile_id} must declare its limitations")
        go = os.environ.get("GO", "go")
        imports = subprocess.run(
            [go, "list", "-buildvcs=false", "-deps", "-f", "{{.ImportPath}}", *["./" + item for item in profile["runtime_entrypoints"]]],
            cwd=ROOT, text=True, capture_output=True,
        )
        if imports.returncode:
            fail(f"composition profile {profile_id} package graph failed: {imports.stderr.strip()}")
        ownership = load_json(MODULES / "package-ownership.json").get("owners", [])
        for package in imports.stdout.splitlines():
            if package.startswith(("synora/internal/", "synora/pkg/")):
                owner = owner_for(package, ownership)
                if owner not in selected:
                    fail(f"composition profile {profile_id} imports excluded module package {package} ({owner})")


def owner_for(package: str, owners: list[dict]) -> str | None:
    matches = []
    for entry in owners:
        prefix = entry["prefix"].rstrip("/")
        match = package == prefix or (entry.get("recursive") and package.startswith(prefix + "/"))
        if match:
            matches.append((len(prefix), entry["module"]))
    if not matches:
        return None
    matches.sort(reverse=True)
    if len(matches) > 1 and matches[0][0] == matches[1][0]:
        fail(f"ambiguous owner for {package}")
    return matches[0][1]


def package_graph() -> dict[str, set[str]]:
    go = os.environ.get("GO", "go")
    result = subprocess.run(
        [go, "list", "-buildvcs=false", "-f", "{{.ImportPath}}|{{range .Imports}}{{.}},{{end}}", "./..."],
        cwd=ROOT, text=True, capture_output=True,
    )
    if result.returncode:
        fail("go list failed: " + result.stderr.strip())
    graph: dict[str, set[str]] = {}
    for line in result.stdout.splitlines():
        package, _, imports = line.partition("|")
        graph[package] = {item for item in imports.split(",") if item}
    return graph


def validate_packages(
    descriptors: dict[str, dict], declared_graph: dict[str, set[str]]
) -> tuple[dict[str, str], dict[str, set[str]]]:
    ownership = load_json(MODULES / "package-ownership.json")
    owners = ownership.get("owners", [])
    graph = package_graph()
    assignments: dict[str, str] = {}
    for package in graph:
        if not package.startswith(("synora/internal/", "synora/pkg/")):
            continue
        owner = owner_for(package, owners)
        if owner is None:
            fail(f"Go package has no module owner: {package}")
        if owner not in descriptors:
            fail(f"package owner is not registered: {package} -> {owner}")
        assignments[package] = owner
    for module_id, descriptor in descriptors.items():
        for source in descriptor["source_packages"]:
            if not source.startswith(("internal/", "pkg/")):
                continue
            package = "synora/" + source
            if package not in graph:
                fail(f"descriptor lists a missing Go package: {module_id}: {source}")
            if assignments.get(package) != module_id:
                fail(f"descriptor/package ownership mismatch: {module_id} lists {source}, owner is {assignments.get(package)}")

    if assignments.get("synora/internal/foundation/contracts") != "synora.foundation":
        fail("synora/internal/foundation/contracts must be owned by synora.foundation")
    if assignments.get("synora/internal/foundation/statestore") != "synora.foundation":
        fail("synora/internal/foundation/statestore must be the Foundation-owned Store implementation")
    if assignments.get("synora/internal/cognitivecore") != "synora.intelligence":
        fail("synora/internal/cognitivecore must remain Intelligence-owned")

    # The CPU MLP implementation is an Intelligence adapter over Core
    # contracts. Keep the dependency one-way so orchestration never depends on
    # a concrete inference backend.
    mlp_package = "synora/internal/intelligence/mlp"
    core_package = "synora/internal/cognitivecore"
    if assignments.get(mlp_package) != "synora.intelligence":
        fail("synora/internal/intelligence/mlp must be Intelligence-owned")
    if mlp_package not in graph or core_package not in graph:
        fail("Intelligence MLP adapter or Cognitive Core package is missing")
    if core_package not in graph[mlp_package]:
        fail("Intelligence MLP adapter must consume Cognitive Core contracts")
    if mlp_package in graph[core_package]:
        fail("Cognitive Core must not import the concrete Intelligence MLP adapter")

    state_encoder_package = "synora/internal/intelligence/stateencoder"
    if assignments.get(state_encoder_package) != "synora.intelligence":
        fail("synora/internal/intelligence/stateencoder must be Intelligence-owned")
    if state_encoder_package not in graph:
        fail("Intelligence state encoder package is missing")
    if any(item.startswith(("synora/internal/", "synora/pkg/")) for item in graph[state_encoder_package]):
        fail("Intelligence state encoder must remain isolated from production contracts and runtime packages")

    automation_package = "synora/internal/automation"
    integration_package = "synora/internal/discovery"
    if assignments.get(automation_package) != "synora.automation":
        fail("synora/internal/automation must be Automation-owned")
    if automation_package not in graph or integration_package not in graph:
        fail("Automation dispatcher or Integration composition package is missing")
    if automation_package not in graph[integration_package]:
        fail("Integration must consume the Automation action-dispatch boundary")
    if any(item.startswith("synora/internal/discovery") for item in graph[automation_package]):
        fail("Automation must not depend on the Integration Discovery implementation")
    if "synora.automation" not in declared_graph.get("synora.integration", set()):
        fail("Integration must declare its Automation dependency")
    integration_descriptor = descriptors.get("synora.integration", {})
    if "synora.platform" not in declared_graph.get("synora.integration", set()):
        fail("Integration must declare its Platform adapter dependency")
    if "synora.platform.network/v1" not in integration_descriptor.get("consumed_interfaces", []):
        fail("Integration must declare consumption of the Platform network interface")

    # Logical source modules may depend on their own domain and Foundation.
    # Integration is the composition edge and is allowed to consume adapters.
    forbidden = {
        "synora.foundation": {"synora.intelligence", "synora.vision", "synora.voice", "synora.automation", "synora.integration", "synora.platform"},
        "synora.intelligence": {"synora.vision", "synora.voice", "synora.automation", "synora.integration", "synora.platform"},
        "synora.vision": {"synora.intelligence", "synora.voice", "synora.automation", "synora.integration", "synora.platform"},
        "synora.automation": {"synora.vision", "synora.voice", "synora.integration", "synora.platform"},
        "synora.voice": {"synora.vision", "synora.automation", "synora.integration", "synora.platform"},
        "synora.platform": {"synora.intelligence", "synora.vision", "synora.voice", "synora.automation", "synora.integration"},
    }
    for source, imports in graph.items():
        source_owner = assignments.get(source)
        if not source_owner:
            continue
        for imported in imports:
            if source in {"synora/pkg/contract", "synora/pkg/contracts", "synora/internal/foundation/contracts"}:
                if imported.startswith("synora/") or "." in imported.split("/", 1)[0]:
                    fail(f"shared contracts may import only the Go standard library: {source} imports {imported}")
                if imported in {"os", "os/exec", "syscall", "path/filepath", "runtime"}:
                    fail(f"shared contract imports operating-system detail: {source} imports {imported}")
            target_owner = assignments.get(imported)
            if target_owner in forbidden.get(source_owner, set()):
                fail(f"forbidden module dependency: {source} ({source_owner}) imports {imported} ({target_owner})")
            if target_owner and target_owner != source_owner and target_owner not in declared_graph[source_owner]:
                fail(f"undeclared module dependency: {source_owner} imports {target_owner} through {source} -> {imported}")

    # Assert that an import cycle cannot be introduced by an internal package.
    active: list[str] = []
    visited: set[str] = set()

    def walk(package: str) -> None:
        if package in active:
            fail("Go package dependency cycle: " + " -> ".join(active[active.index(package):] + [package]))
        if package in visited:
            return
        active.append(package)
        for dependency in sorted(graph.get(package, set())):
            if dependency in graph:
                walk(dependency)
        active.pop()
        visited.add(package)

    for package in sorted(graph):
        walk(package)
    edges = {module_id: set() for module_id in descriptors}
    for source, imports in graph.items():
        source_owner = assignments.get(source)
        if not source_owner:
            continue
        for imported in imports:
            target_owner = assignments.get(imported)
            if target_owner and source_owner != target_owner:
                edges[source_owner].add(target_owner)
    return assignments, edges


def validate_vision_runtime_boundary() -> None:
    """Keep the concrete RKNN runtime import inside its one adapter module."""
    allowed = (VISION_WORKER / "core" / "rknn_backend.py").resolve()
    for source in VISION_WORKER.rglob("*.py"):
        if "tests" in source.parts:
            continue
        try:
            tree = ast.parse(source.read_text(), filename=str(source))
        except (OSError, SyntaxError) as exc:
            fail(f"cannot inspect Vision Python source {source.relative_to(ROOT)}: {exc}")
        for node in ast.walk(tree):
            imported = []
            if isinstance(node, ast.Import):
                imported.extend(alias.name for alias in node.names)
            elif isinstance(node, ast.ImportFrom):
                imported.append(node.module or "")
                imported.extend(alias.name for alias in node.names)
            if any(name == "rknnlite" or name.startswith("rknnlite.") for name in imported):
                if source.resolve() != allowed:
                    fail(f"RKNN runtime import escaped adapter: {source.relative_to(ROOT)}")


def validate_platform_command_boundary() -> None:
    """Keep network host-command execution in its Platform adapter."""
    network_package = ROOT / "internal" / "discovery" / "network"
    adapter = network_package / "command.go"
    if not adapter.exists():
        fail("Platform network command adapter is missing")
    for source in network_package.glob("*.go"):
        if source.name.endswith("_test.go"):
            continue
        text = source.read_text()
        if source.resolve() != adapter.resolve() and (
            '"os/exec"' in text or "exec.Command(" in text
        ):
            fail(f"host command invocation escaped Platform adapter: {source.relative_to(ROOT)}")
    adapter_text = adapter.read_text()
    if '"os/exec"' not in adapter_text or "type OSCommandExecutor" not in adapter_text:
        fail("Platform command adapter must own the OS exec implementation")


def main() -> None:
    descriptors, declared_graph = validate_descriptors()
    validate_composition_profiles(descriptors)
    assignments, actual_edges = validate_packages(descriptors, declared_graph)
    validate_vision_runtime_boundary()
    validate_platform_command_boundary()
    print(
        "architecture check: PASS "
        f"modules={len(descriptors)} owned_go_packages={len(assignments)} "
        f"declared_edges={sum(map(len, declared_graph.values()))} "
        "python_rknn_imports=isolated "
        f"actual_module_edges={sum(map(len, actual_edges.values()))}"
    )


if __name__ == "__main__":
    main()
