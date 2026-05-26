#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/.." && pwd)

CONFIG_FILE="${CONFIG_FILE:-${REPO_ROOT}/cliff.toml}"
CHANGELOG_FILE="${CHANGELOG_FILE:-${REPO_ROOT}/CHANGELOG.md}"
GIT_CLIFF_VERSION="${GIT_CLIFF_VERSION:-2.13.1}"
FETCH_TAGS="${FETCH_TAGS:-true}"

MODE="release"
TARGET_TAG=""
PREVIOUS_TAG=""

usage() {
  cat <<'EOF'
Usage:
  hack/changelog-generate.sh --bootstrap [--output <path>]
  hack/changelog-generate.sh --tag <vX.Y.Z> [--previous-tag <vX.Y.Z>] [--output <path>]

Options:
  --bootstrap             Generate the full historical changelog from all stable tags.
  --tag                   Generate and prepend a single release entry for the given stable tag.
  --previous-tag          Override the automatically detected previous stable tag.
  --output                Path to the changelog file. Defaults to CHANGELOG.md in the repo root.
  --no-fetch-tags         Skip 'git fetch --tags --force'.
  --help                  Show this help text.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --bootstrap)
      MODE="bootstrap"
      ;;
    --tag)
      TARGET_TAG="${2:-}"
      shift
      ;;
    --previous-tag)
      PREVIOUS_TAG="${2:-}"
      shift
      ;;
    --output)
      CHANGELOG_FILE="${2:-}"
      shift
      ;;
    --no-fetch-tags)
      FETCH_TAGS="false"
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
  shift
done

if [[ ! -f "${CONFIG_FILE}" ]]; then
  echo "Unable to find git-cliff config: ${CONFIG_FILE}" >&2
  exit 1
fi

stable_tag_pattern='^v[0-9]+\.[0-9]+\.[0-9]+$'

if [[ "${FETCH_TAGS}" == "true" ]]; then
  git -C "${REPO_ROOT}" fetch --tags --force >/dev/null 2>&1 || true
fi

download_git_cliff() {
  local os arch target archive_name download_url cache_root extract_dir archive_path

  case "$(uname -s)" in
    Linux) os="unknown-linux-gnu" ;;
    Darwin) os="apple-darwin" ;;
    *)
      echo "Unsupported OS for auto-installing git-cliff: $(uname -s)" >&2
      return 1
      ;;
  esac

  case "$(uname -m)" in
    x86_64|amd64) arch="x86_64" ;;
    aarch64|arm64) arch="aarch64" ;;
    *)
      echo "Unsupported architecture for auto-installing git-cliff: $(uname -m)" >&2
      return 1
      ;;
  esac

  archive_name="git-cliff-${GIT_CLIFF_VERSION}-${arch}-${os}.tar.gz"
  cache_root="${XDG_CACHE_HOME:-${HOME}/.cache}/rosa/git-cliff/${GIT_CLIFF_VERSION}/${arch}-${os}"
  target="${cache_root}/git-cliff"

  if [[ -x "${target}" ]]; then
    printf '%s\n' "${target}"
    return 0
  fi

  mkdir -p "${cache_root}"
  extract_dir=$(mktemp -d)
  archive_path="${extract_dir}/${archive_name}"
  download_url="https://github.com/orhun/git-cliff/releases/download/v${GIT_CLIFF_VERSION}/${archive_name}"

  echo "Downloading git-cliff v${GIT_CLIFF_VERSION}..." >&2
  curl -fsSL -o "${archive_path}" "${download_url}"
  tar -xzf "${archive_path}" -C "${extract_dir}" --strip-components=1 "git-cliff-${GIT_CLIFF_VERSION}/git-cliff"
  install -m 0755 "${extract_dir}/git-cliff" "${target}"
  rm -rf "${extract_dir}"

  printf '%s\n' "${target}"
}

resolve_git_cliff() {
  if [[ -n "${GIT_CLIFF_BIN:-}" && -x "${GIT_CLIFF_BIN}" ]]; then
    printf '%s\n' "${GIT_CLIFF_BIN}"
    return 0
  fi

  if command -v git-cliff >/dev/null 2>&1; then
    command -v git-cliff
    return 0
  fi

  download_git_cliff
}

mapfile -t stable_tags < <(git -C "${REPO_ROOT}" tag -l 'v*' | awk '/^v[0-9]+\.[0-9]+\.[0-9]+$/' | sort -V)

if [[ "${#stable_tags[@]}" -eq 0 ]]; then
  echo "No stable tags found in ${REPO_ROOT}" >&2
  exit 1
fi

git_cliff_bin=$(resolve_git_cliff)

if [[ "${MODE}" == "bootstrap" ]]; then
  : > "${CHANGELOG_FILE}"
  for i in "${!stable_tags[@]}"; do
    current_tag="${stable_tags[$i]}"
    if (( i == 0 )); then
      commit_range="${current_tag}"
    else
      previous_bootstrap_tag="${stable_tags[$((i - 1))]}"
      commit_range="${previous_bootstrap_tag}..${current_tag}"
    fi

    "${git_cliff_bin}" \
      --config "${CONFIG_FILE}" \
      --tag "${current_tag#v}" \
      "${commit_range}" \
      --prepend "${CHANGELOG_FILE}" >/dev/null
  done

  exit 0
fi

if [[ -z "${TARGET_TAG}" ]]; then
  echo "--tag is required unless --bootstrap is used" >&2
  usage >&2
  exit 1
fi

if ! [[ "${TARGET_TAG}" =~ ${stable_tag_pattern} ]]; then
  echo "Tag '${TARGET_TAG}' does not match the expected stable format vX.Y.Z" >&2
  exit 1
fi

if ! git -C "${REPO_ROOT}" rev-parse --verify "${TARGET_TAG}^{tag}" >/dev/null 2>&1; then
  echo "Tag '${TARGET_TAG}' was not found locally" >&2
  exit 1
fi

if [[ -z "${PREVIOUS_TAG}" ]]; then
  previous_index=-1
  for i in "${!stable_tags[@]}"; do
    if [[ "${stable_tags[$i]}" == "${TARGET_TAG}" ]]; then
      previous_index=$((i - 1))
      break
    fi
  done

  if (( previous_index < 0 )); then
    echo "Unable to determine a previous stable tag for '${TARGET_TAG}'" >&2
    exit 1
  fi

  PREVIOUS_TAG="${stable_tags[$previous_index]}"
fi

if ! [[ "${PREVIOUS_TAG}" =~ ${stable_tag_pattern} ]]; then
  echo "Previous tag '${PREVIOUS_TAG}' does not match the expected stable format vX.Y.Z" >&2
  exit 1
fi

if ! git -C "${REPO_ROOT}" rev-parse --verify "${PREVIOUS_TAG}^{tag}" >/dev/null 2>&1; then
  echo "Previous tag '${PREVIOUS_TAG}' was not found locally" >&2
  exit 1
fi

commit_count=$(git -C "${REPO_ROOT}" rev-list --count "${PREVIOUS_TAG}..${TARGET_TAG}")
if [[ "${commit_count}" == "0" ]]; then
  echo "No commits found between ${PREVIOUS_TAG} and ${TARGET_TAG}; nothing to do."
  exit 0
fi

"${git_cliff_bin}" \
  --config "${CONFIG_FILE}" \
  --tag "${TARGET_TAG#v}" \
  "${PREVIOUS_TAG}..${TARGET_TAG}" \
  --prepend "${CHANGELOG_FILE}"
