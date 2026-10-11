#!/usr/bin/env bash
set -euo pipefail

readonly repository_url="git@github.com:ANLEXISS/SynOEMData.git"
readonly source_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
repo_dir="${SYNOEMDATA_WORKTREE:-/home/rock/SynOEMData}"
branch="${SYNOEMDATA_BRANCH:-}"

if [[ "${1:-}" != "--validated" || $# -ne 1 ]]; then
  cat >&2 <<'EOF'
Usage: tools/sync-documentation.sh --validated

Run only after the milestone implementation is complete, required validation
has passed or its limitations are explicitly recorded, and milestone reports
have been updated. GitHub pushes use the existing SSH authentication.
EOF
  exit 2
fi

if [[ ! -d "$source_root/docs" ]]; then
  echo "ERROR: source documentation directory not found: $source_root/docs" >&2
  exit 2
fi

if [[ ! -d "$repo_dir/.git" ]]; then
  if [[ -e "$repo_dir" ]]; then
    echo "ERROR: destination exists but is not a Git worktree: $repo_dir" >&2
    exit 2
  fi
  git clone "$repository_url" "$repo_dir"
fi

origin_url="$(git -C "$repo_dir" remote get-url origin)"
case "$origin_url" in
  "$repository_url"|"https://github.com/ANLEXISS/SynOEMData.git") ;;
  *)
    echo "ERROR: origin does not match ANLEXISS/SynOEMData: $origin_url" >&2
    exit 2
    ;;
esac

if [[ -z "$branch" ]]; then
  remote_head="$(git -C "$repo_dir" symbolic-ref -q --short refs/remotes/origin/HEAD || true)"
  if [[ -n "$remote_head" ]]; then
    branch="${remote_head#origin/}"
  else
    branch="main"
  fi
fi

if [[ -n "$(git -C "$repo_dir" status --porcelain)" ]]; then
  echo "ERROR: documentation worktree has local changes; preserve or resolve them before synchronization." >&2
  exit 2
fi

git -C "$repo_dir" fetch origin
if git -C "$repo_dir" show-ref --verify --quiet "refs/heads/$branch"; then
  git -C "$repo_dir" switch "$branch"
elif [[ "$(git -C "$repo_dir" symbolic-ref -q --short HEAD || true)" != "$branch" ]]; then
  git -C "$repo_dir" switch -c "$branch"
fi

if git -C "$repo_dir" show-ref --verify --quiet "refs/remotes/origin/$branch"; then
  git -C "$repo_dir" merge --ff-only "origin/$branch"
fi

staging="$(mktemp -d "${TMPDIR:-/tmp}/synora-doc-sync.XXXXXX")"
trap 'rm -rf "$staging"' EXIT
mkdir -p "$staging/docs"

discovered=0
approved=0
excluded=0
while IFS= read -r -d '' source_file; do
  ((discovered += 1))
  relative="${source_file#"$source_root/docs/"}"
  case "$relative" in
    security-auth.md)
      echo "EXCLUDED docs/$relative — contains a resident face-profile example with personal identity and home-zone data."
      ((excluded += 1))
      continue
      ;;
    security-network.md)
      echo "EXCLUDED docs/$relative — contains sensitive network/pairing design and unresolved security details."
      ((excluded += 1))
      continue
      ;;
  esac

  case "$source_file" in
    *.md) ;;
    *)
      echo "ERROR: only Markdown reports may be synchronized: $source_file" >&2
      exit 2
      ;;
  esac

  mkdir -p "$staging/docs/$(dirname "$relative")"
  cp -p "$source_file" "$staging/docs/$relative"
  ((approved += 1))
done < <(find "$source_root/docs" -type f -print0)

if (( discovered == 0 )); then
  echo "ERROR: no documentation files were discovered." >&2
  exit 2
fi

if find "$staging/docs" -type f ! -name '*.md' -print -quit | grep -q .; then
  echo "ERROR: staged content includes a non-Markdown file." >&2
  exit 2
fi

secret_pattern='-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----|gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{30,}|AKIA[0-9A-Z]{16}|Bearer[[:space:]]+[A-Za-z0-9._~-]{24,}|(password|passwd|secret|token|api[_-]?key)[[:space:]_-]*[:=][[:space:]]*'
if grep -RInE -- "$secret_pattern" "$staging/docs"; then
  echo "ERROR: secret scan failed; nothing was copied or committed." >&2
  exit 3
fi

manifest="$repo_dir/.synoraoem-docs-manifest"
manifest_tmp="$staging/manifest"
if [[ -f "$manifest" ]]; then
  cat "$manifest" > "$manifest_tmp"
else
  : > "$manifest_tmp"
fi

while IFS= read -r -d '' staged_file; do
  relative="${staged_file#"$staging/"}"
  destination="$repo_dir/$relative"
  if [[ -e "$destination" ]] && ! grep -Fxq -- "$relative" "$manifest_tmp" && ! cmp -s "$staged_file" "$destination"; then
    echo "ERROR: refusing to overwrite unowned destination content: $relative" >&2
    exit 4
  fi
  printf '%s\n' "$relative" >> "$manifest_tmp"
done < <(find "$staging/docs" -type f -print0)

sort -u "$manifest_tmp" -o "$manifest_tmp"
rsync -a "$staging/docs/" "$repo_dir/docs/"
cp "$manifest_tmp" "$manifest"

while IFS= read -r -d '' staged_file; do
  relative="${staged_file#"$staging/"}"
  git -C "$repo_dir" add -- "$relative"
done < <(find "$staging/docs" -type f -print0)
git -C "$repo_dir" add -- .synoraoem-docs-manifest

if ! git -C "$repo_dir" diff --cached --quiet; then
  git -C "$repo_dir" commit -m "docs: synchronize SynoraOEM reports"
fi

if ! git -C "$repo_dir" push -u origin "$branch"; then
  echo "SYNC=PENDING; reports and commit are preserved locally at $repo_dir." >&2
  exit 5
fi

commit="$(git -C "$repo_dir" rev-parse HEAD)"
printf 'DISCOVERED=%s APPROVED=%s EXCLUDED=%s\n' "$discovered" "$approved" "$excluded"
printf 'REPOSITORY=https://github.com/ANLEXISS/SynOEMData\nBRANCH=%s\nCOMMIT=%s\nSYNC=PUBLISHED\n' "$branch" "$commit"
