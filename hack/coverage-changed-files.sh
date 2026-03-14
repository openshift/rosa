#!/usr/bin/env bash

set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

required_coverage_percent="80"
gocovdiff_module="github.com/vearutop/gocovdiff"
gocovdiff_version="v1.4.2"

diff_base_args=(--cached)
mapfile -t candidate_files < <(git diff "${diff_base_args[@]}" --name-only --diff-filter=ACMR -- '*.go')
if [ "${#candidate_files[@]}" -eq 0 ]; then
  diff_base_args=()
  mapfile -t candidate_files < <(git diff --name-only --diff-filter=ACMR -- '*.go')
fi

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/rosa-changed-coverage-XXXXXX")
coverage_profile="$tmp_dir/cover.out"
diff_file="$tmp_dir/changes.diff"
delta_file="$tmp_dir/delta-cov.txt"
trap 'rm -rf "$tmp_dir"' EXIT

git diff "${diff_base_args[@]}" -U0 -- '*.go' > "$diff_file"
if [ ! -s "$diff_file" ]; then
  exit 0
fi

declare -A package_seen=()
declare -a changed_packages=()
for file_path in "${candidate_files[@]}"; do
  [ -z "$file_path" ] && continue
  case "$file_path" in
    vendor/*|.tmp/*|*_test.go)
      continue
      ;;
  esac

  [ -f "$file_path" ] || continue
  package_name=$(go list "./$(dirname "$file_path")" 2>/dev/null || true)
  if [ -n "$package_name" ] && [ -z "${package_seen[$package_name]+x}" ]; then
    package_seen["$package_name"]=1
    changed_packages+=("$package_name")
  fi
done

if [ "${#changed_packages[@]}" -eq 0 ]; then
  exit 0
fi

go test -count=1 -covermode=atomic -coverprofile="$coverage_profile" "${changed_packages[@]}"

GOFLAGS='-mod=mod' go run "${gocovdiff_module}@${gocovdiff_version}" \
  -diff "$diff_file" \
  -cov "$coverage_profile" \
  -exclude "vendor/,.tmp/" \
  -target-delta-cov "$required_coverage_percent" \
  -delta-cov-file "$delta_file"

if [ -s "$delta_file" ]; then
  cat "$delta_file"
  echo
fi

if grep -q "coverage is less than" "$delta_file"; then
  exit 1
fi
