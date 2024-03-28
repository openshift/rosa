#!/usr/bin/env bash

# This regex matches the pull number with an author from the job spec. Prow does not have a PULL_AUTHOR job var
PULL_AUTHOR=$(echo $JOB_SPEC | grep -Po "$PULL_NUMBER,\"author\":\"\K[^\"]*")

# PULL_BASE_SHA and PULL_PULL_SHA are set in prow job
PULL_BASE_SHA="${PULL_BASE_SHA:-$(git merge-base master HEAD)}"
PULL_PULL_SHA="${PULL_PULL_SHA:-HEAD}"

# Lists of GitHub users for whom the commit validation should be skipped:
# # This contains the bots used in the ROSA project
# * openshift-merge-bot[bot] - bot responsible for merigng MRs
# * openshift-ci[bot] - bot responsible for running tests / approvals
declare -a skip_pr_authors=(
  "openshift-ci[bot]"
  "openshift-merge-bot[bot]"
)
echo "The PR Author is \"${PULL_AUTHOR}\""
for skip_pr_author in "${skip_pr_authors[@]}"
do
  if [ "${PULL_AUTHOR}" = "${skip_pr_author}" ]; then
    echo "The commits created by this PR author (probably bot) should be skipped!!!"
    exit 0
  fi
done

pattern="^(Revert \")?[A-Z]+-[0-9]+ \| (feat|fix|docs|style|refactor|test|chore|build|ci|perf): .*$"

commits=$(git rev-list --no-merges $PULL_BASE_SHA..$PULL_PULL_SHA)

for sha in $commits; do
  message=$(git log --format=%B -n 1 $sha)
  if ! [[ $message =~ $pattern ]]; then
    echo "Invalid commit message: $message"
    echo "Expected format: JIRA_TICKET | TYPE: MESSAGE"
    echo "Please check CONTRIBUTING.md for more details"
    exit 1
  fi
done

echo "All commit messages are valid"
