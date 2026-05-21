#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/.." && pwd)

CHANGELOG_FILE="${CHANGELOG_FILE:-${REPO_ROOT}/CHANGELOG.md}"
REMOTE_NAME="${CHANGELOG_REMOTE:-origin}"
PR_BRANCH_PREFIX="${CHANGELOG_PR_BRANCH_PREFIX:-changelog}"
JIRA_KEY="${CHANGELOG_JIRA_KEY:-OCM-00000}"
COMMIT_TYPE="${CHANGELOG_COMMIT_TYPE:-docs}"
AUTHOR_NAME="${CHANGELOG_AUTHOR_NAME:-OpenShift CI Bot}"
AUTHOR_EMAIL="${CHANGELOG_AUTHOR_EMAIL:-ci-bot@redhat.com}"
BASE_BRANCH="${CHANGELOG_BASE_BRANCH:-master}"

TARGET_TAG=""
PREVIOUS_TAG=""

usage() {
  cat <<'EOF'
Usage:
  hack/changelog-pr.sh [--tag <vX.Y.Z>] [--previous-tag <vX.Y.Z>]

Environment:
  GITHUB_TOKEN                   Token used to push the changelog branch and create/update the PR.
  CHANGELOG_REMOTE               Remote to push to. Defaults to origin.
  CHANGELOG_PR_BRANCH_PREFIX     Prefix for the generated changelog branch. Defaults to "changelog".
  CHANGELOG_JIRA_KEY             Jira key used in the generated commit/PR title. Defaults to OCM-00000.
  CHANGELOG_COMMIT_TYPE          Commit type used in the generated commit/PR title. Defaults to docs.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --tag)
      TARGET_TAG="${2:-}"
      shift
      ;;
    --previous-tag)
      PREVIOUS_TAG="${2:-}"
      shift
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

if [[ -z "${GITHUB_TOKEN:-}" ]]; then
  echo "GITHUB_TOKEN must be set to push the changelog branch and create a PR" >&2
  exit 1
fi

if [[ -z "${TARGET_TAG}" ]]; then
  TARGET_TAG=$(git -C "${REPO_ROOT}" describe --exact-match --tags HEAD 2>/dev/null || true)
fi

if [[ -z "${TARGET_TAG}" ]]; then
  echo "Unable to determine the stable tag for the current revision; pass --tag explicitly" >&2
  exit 1
fi

git -C "${REPO_ROOT}" config user.name "${AUTHOR_NAME}"
git -C "${REPO_ROOT}" config user.email "${AUTHOR_EMAIL}"

generate_args=( --tag "${TARGET_TAG}" )
if [[ -n "${PREVIOUS_TAG}" ]]; then
  generate_args+=( --previous-tag "${PREVIOUS_TAG}" )
fi
"${SCRIPT_DIR}/changelog-generate.sh" "${generate_args[@]}"

if git -C "${REPO_ROOT}" diff --quiet -- "${CHANGELOG_FILE}"; then
  echo "No changelog changes detected; no PR will be created."
  exit 0
fi

branch_name="${PR_BRANCH_PREFIX}-${TARGET_TAG}"
branch_name="${branch_name//\//-}"
commit_subject="${JIRA_KEY} | ${COMMIT_TYPE}: add changelog for ${TARGET_TAG}"

git -C "${REPO_ROOT}" checkout -B "${branch_name}"
git -C "${REPO_ROOT}" add "${CHANGELOG_FILE}"
git -C "${REPO_ROOT}" commit -m "${commit_subject}"
git -C "${REPO_ROOT}" \
  -c "credential.helper=!f() { echo username=x-access-token; echo password=${GITHUB_TOKEN}; }; f" \
  push --force-with-lease "${REMOTE_NAME}" "${branch_name}"

remote_url=$(git -C "${REPO_ROOT}" remote get-url "${REMOTE_NAME}")
if [[ "${remote_url}" =~ github\.com[:/]([^/]+/[^/.]+)(\.git)?$ ]]; then
  repo_slug="${BASH_REMATCH[1]}"
else
  echo "Unable to determine GitHub repo slug from remote '${REMOTE_NAME}': ${remote_url}" >&2
  exit 1
fi

repo_owner="${repo_slug%%/*}"
repo_name="${repo_slug##*/}"
api_base="https://api.github.com/repos/${repo_owner}/${repo_name}"

pr_title="${commit_subject}"
pr_body=$(cat <<EOF
## PR Summary

Update the historical \`CHANGELOG.md\` entry for ${TARGET_TAG} using the repository changelog automation.

## Detailed Description of the Issue

This PR was generated after the ${TARGET_TAG} tag to keep the repository's historical changelog in sync with the released commit range.

## Related Issues and PRs

- JIRA: [${JIRA_KEY}](https://issues.redhat.com/browse/${JIRA_KEY})
- Fixes: \`N/A\`
- Related PR(s): N/A
- Related design/docs: N/A

## Type of Change

- [ ] feat - adds a new user-facing capability.
- [ ] fix - resolves an incorrect behavior or bug.
- [x] docs - updates documentation only.
- [ ] style - formatting or naming changes with no logic impact.
- [ ] refactor - code restructuring with no behavior change.
- [ ] test - adds or updates tests only.
- [ ] chore - maintenance work (tooling, housekeeping, non-product code).
- [ ] build - changes build system, packaging, or dependencies for build output.
- [ ] ci - changes CI pipelines, jobs, or automation workflows.
- [ ] perf - improves performance without changing intended behavior.

## Previous Behavior

The repository historical changelog did not yet include an entry for ${TARGET_TAG}.

## Behavior After This Change

\`CHANGELOG.md\` includes the historical entry for ${TARGET_TAG}.

## How to Test (Step-by-Step)

### Preconditions

- Tag ${TARGET_TAG} exists in the repository.

### Test Steps

1. Inspect the generated \`CHANGELOG.md\` entry for ${TARGET_TAG}.
2. Verify the commit grouping and formatting match the repository changelog conventions.
3. Confirm the entry covers the commit range for the release.

### Expected Results

The historical changelog contains a correctly formatted entry for ${TARGET_TAG}.

## Proof of the Fix

- Screenshots: N/A
- Videos: N/A
- Logs/CLI output: Generated changelog diff in this PR
- Other artifacts: N/A

## Breaking Changes

- [x] No breaking changes
- [ ] Yes, this PR introduces a breaking change (describe impact and migration plan below)

### Breaking Change Details / Migration Plan

N/A

## Developer Verification Checklist

- [x] Commit subject/title follows \`[JIRA-TICKET] | [TYPE]: <MESSAGE>\`.
- [x] PR description clearly explains both **what** changed and **why**.
- [x] Relevant Jira/GitHub issues and related PRs are linked.
- [ ] \`make install-hooks\` has been run in this clone.
- [ ] Tests were added/updated where appropriate.
- [ ] I manually tested the change.
- [ ] \`make test\` passes.
- [ ] \`make lint\` passes.
- [ ] \`make rosa\` passes.
- [x] Documentation or repo-local agent guidance was added/updated where appropriate.
- [x] Any risk, limitation, or follow-up work is documented.
EOF
)

existing_pr_number=$(curl -fsSL \
  -H "Authorization: Bearer ${GITHUB_TOKEN}" \
  -H "Accept: application/vnd.github+json" \
  "${api_base}/pulls?head=${repo_owner}:${branch_name}&state=open" \
  | jq -r '.[0].number // empty')

if [[ -n "${existing_pr_number}" ]]; then
  curl -fsSL \
    -X PATCH \
    -H "Authorization: Bearer ${GITHUB_TOKEN}" \
    -H "Accept: application/vnd.github+json" \
    "${api_base}/pulls/${existing_pr_number}" \
    -d "$(jq -n --arg title "${pr_title}" --arg body "${pr_body}" '{title: $title, body: $body}')" >/dev/null
  echo "Updated existing PR #${existing_pr_number}"
  exit 0
fi

pr_url=$(curl -fsSL \
  -X POST \
  -H "Authorization: Bearer ${GITHUB_TOKEN}" \
  -H "Accept: application/vnd.github+json" \
  "${api_base}/pulls" \
  -d "$(jq -n --arg title "${pr_title}" --arg head "${branch_name}" --arg base "${BASE_BRANCH}" --arg body "${pr_body}" '{title: $title, head: $head, base: $base, body: $body, maintainer_can_modify: false}')" \
  | jq -r '.html_url // empty')

if [[ -z "${pr_url}" ]]; then
  echo "Failed to create changelog PR" >&2
  exit 1
fi

echo "Created PR: ${pr_url}"
